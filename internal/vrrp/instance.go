package vrrp

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/loveuer/go-alived/pkg/logger"
	"github.com/loveuer/go-alived/pkg/netif"
)

type Instance struct {
	Name            string
	VirtualRouterID uint8
	Priority        uint8
	AdvertInterval  uint8
	Interface       string
	VirtualIPs      []net.IP
	AuthType        uint8
	AuthPass        string
	TrackScripts    []string

	state           *StateMachine
	priorityCalc    *PriorityCalculator
	history         *StateHistory
	socket          *Socket
	arpSender       *ARPSender
	netInterface    *netif.Interface

	advertTimer     *Timer
	masterDownTimer *Timer

	running bool
	stopCh  chan struct{}
	wg      sync.WaitGroup
	mu      sync.RWMutex

	log *logger.Logger

	onMaster func()
	onBackup func()
	onFault  func()
}

func NewInstance(
	name string,
	vrID uint8,
	priority uint8,
	advertInt uint8,
	iface string,
	vips []string,
	authType string,
	authPass string,
	trackScripts []string,
	log *logger.Logger,
) (*Instance, error) {
	if vrID < 1 || vrID > 255 {
		return nil, fmt.Errorf("invalid virtual router ID: %d", vrID)
	}

	if priority < 1 || priority > 255 {
		return nil, fmt.Errorf("invalid priority: %d", priority)
	}

	virtualIPs := make([]net.IP, 0, len(vips))
	for _, vip := range vips {
		ip, _, err := net.ParseCIDR(vip)
		if err != nil {
			return nil, fmt.Errorf("invalid VIP %s: %w", vip, err)
		}
		virtualIPs = append(virtualIPs, ip)
	}

	var authTypeNum uint8
	switch authType {
	case "NONE", "":
		authTypeNum = AuthTypeNone
	case "PASS":
		authTypeNum = AuthTypeSimpleText
	default:
		return nil, fmt.Errorf("unsupported auth type: %s", authType)
	}

	netInterface, err := netif.GetInterface(iface)
	if err != nil {
		return nil, fmt.Errorf("failed to get interface: %w", err)
	}

	inst := &Instance{
		Name:            name,
		VirtualRouterID: vrID,
		Priority:        priority,
		AdvertInterval:  advertInt,
		Interface:       iface,
		VirtualIPs:      virtualIPs,
		AuthType:        authTypeNum,
		AuthPass:        authPass,
		TrackScripts:    trackScripts,
		state:           NewStateMachine(StateInit),
		priorityCalc:    NewPriorityCalculator(priority),
		history:         NewStateHistory(100),
		netInterface:    netInterface,
		stopCh:          make(chan struct{}),
		log:             log,
	}

	inst.advertTimer = NewTimer(time.Duration(advertInt)*time.Second, inst.onAdvertTimer)
	inst.masterDownTimer = NewTimer(CalculateMasterDownInterval(advertInt), inst.onMasterDownTimer)

	inst.state.OnStateChange(func(old, new State) {
		inst.history.Add(old, new, "state transition")
		inst.log.Info("[%s] state changed: %s -> %s", inst.Name, old, new)
		inst.handleStateChange(old, new)
	})

	return inst, nil
}

func (inst *Instance) Start() error {
	inst.mu.Lock()
	if inst.running {
		inst.mu.Unlock()
		return fmt.Errorf("instance %s already running", inst.Name)
	}
	inst.running = true
	inst.mu.Unlock()

	var err error
	inst.socket, err = NewSocket(inst.Interface)
	if err != nil {
		return fmt.Errorf("failed to create socket: %w", err)
	}

	inst.arpSender, err = NewARPSender(inst.Interface)
	if err != nil {
		inst.socket.Close()
		return fmt.Errorf("failed to create ARP sender: %w", err)
	}

	inst.log.Info("[%s] starting VRRP instance (VRID=%d, Priority=%d, Interface=%s)",
		inst.Name, inst.VirtualRouterID, inst.Priority, inst.Interface)

	inst.state.SetState(StateBackup)
	inst.masterDownTimer.Start()

	inst.wg.Add(1)
	go inst.receiveLoop()

	return nil
}

func (inst *Instance) Stop() {
	inst.mu.Lock()
	if !inst.running {
		inst.mu.Unlock()
		return
	}
	inst.running = false
	inst.mu.Unlock()

	inst.log.Info("[%s] stopping VRRP instance", inst.Name)

	close(inst.stopCh)
	inst.wg.Wait()

	inst.advertTimer.Stop()
	inst.masterDownTimer.Stop()

	if inst.state.GetState() == StateMaster {
		inst.removeVIPs()
	}

	if inst.socket != nil {
		inst.socket.Close()
	}

	if inst.arpSender != nil {
		inst.arpSender.Close()
	}

	inst.state.SetState(StateInit)
}

func (inst *Instance) receiveLoop() {
	defer inst.wg.Done()

	for {
		select {
		case <-inst.stopCh:
			return
		default:
		}

		pkt, srcIP, err := inst.socket.Receive()
		if err != nil {
			inst.log.Debug("[%s] failed to receive packet: %v", inst.Name, err)
			continue
		}

		if pkt.VirtualRtrID != inst.VirtualRouterID {
			continue
		}

		if err := pkt.Validate(inst.AuthPass); err != nil {
			inst.log.Warn("[%s] packet validation failed: %v", inst.Name, err)
			continue
		}

		inst.handleAdvertisement(pkt, srcIP)
	}
}

func (inst *Instance) handleAdvertisement(pkt *VRRPPacket, srcIP net.IP) {
	currentState := inst.state.GetState()
	localPriority := inst.priorityCalc.GetPriority()

	inst.log.Debug("[%s] received advertisement from %s (priority=%d, state=%s)",
		inst.Name, srcIP, pkt.Priority, currentState)

	switch currentState {
	case StateBackup:
		if pkt.Priority == 0 {
			inst.masterDownTimer.SetDuration(CalculateSkewTime(localPriority))
			inst.masterDownTimer.Reset()
		} else if !ShouldBecomeMaster(localPriority, pkt.Priority, inst.socket.localIP.String(), srcIP.String()) {
			inst.masterDownTimer.Reset()
		}

	case StateMaster:
		if ShouldBecomeMaster(pkt.Priority, localPriority, srcIP.String(), inst.socket.localIP.String()) {
			inst.log.Warn("[%s] received higher priority advertisement, stepping down", inst.Name)
			inst.state.SetState(StateBackup)
		}
	}
}

func (inst *Instance) onAdvertTimer() {
	if inst.state.GetState() == StateMaster {
		inst.sendAdvertisement()
		inst.advertTimer.Start()
	}
}

func (inst *Instance) onMasterDownTimer() {
	if inst.state.GetState() == StateBackup {
		inst.log.Info("[%s] master down timer expired, becoming master", inst.Name)
		inst.state.SetState(StateMaster)
	}
}

func (inst *Instance) sendAdvertisement() error {
	priority := inst.priorityCalc.GetPriority()

	pkt := NewAdvertisement(
		inst.VirtualRouterID,
		priority,
		inst.AdvertInterval,
		inst.VirtualIPs,
		inst.AuthType,
		inst.AuthPass,
	)

	if err := inst.socket.Send(pkt); err != nil {
		inst.log.Error("[%s] failed to send advertisement: %v", inst.Name, err)
		return err
	}

	inst.log.Debug("[%s] sent advertisement (priority=%d)", inst.Name, priority)
	return nil
}

func (inst *Instance) handleStateChange(old, new State) {
	switch new {
	case StateMaster:
		inst.becomeMaster()
	case StateBackup:
		inst.becomeBackup(old)
	case StateFault:
		inst.becomeFault()
	}
}

func (inst *Instance) becomeMaster() {
	inst.log.Info("[%s] transitioning to MASTER state", inst.Name)

	if err := inst.addVIPs(); err != nil {
		inst.log.Error("[%s] failed to add VIPs: %v", inst.Name, err)
		inst.state.SetState(StateFault)
		return
	}

	if err := inst.arpSender.SendGratuitousARPForIPs(inst.VirtualIPs); err != nil {
		inst.log.Error("[%s] failed to send gratuitous ARP: %v", inst.Name, err)
	}

	inst.masterDownTimer.Stop()
	inst.advertTimer.Start()

	inst.sendAdvertisement()

	if inst.onMaster != nil {
		inst.onMaster()
	}
}

func (inst *Instance) becomeBackup(oldState State) {
	inst.log.Info("[%s] transitioning to BACKUP state", inst.Name)

	inst.advertTimer.Stop()

	if oldState == StateMaster {
		if err := inst.removeVIPs(); err != nil {
			inst.log.Error("[%s] failed to remove VIPs: %v", inst.Name, err)
		}
	}

	inst.masterDownTimer.Reset()

	if inst.onBackup != nil {
		inst.onBackup()
	}
}

func (inst *Instance) becomeFault() {
	inst.log.Error("[%s] transitioning to FAULT state", inst.Name)

	inst.advertTimer.Stop()
	inst.masterDownTimer.Stop()

	if err := inst.removeVIPs(); err != nil {
		inst.log.Error("[%s] failed to remove VIPs: %v", inst.Name, err)
	}

	if inst.onFault != nil {
		inst.onFault()
	}
}

func (inst *Instance) addVIPs() error {
	inst.log.Info("[%s] adding virtual IPs", inst.Name)

	for _, vipStr := range inst.getVIPsWithCIDR() {
		if err := inst.netInterface.AddIP(vipStr); err != nil {
			inst.log.Error("[%s] failed to add VIP %s: %v", inst.Name, vipStr, err)
			return err
		}
		inst.log.Info("[%s] added VIP %s", inst.Name, vipStr)
	}

	return nil
}

func (inst *Instance) removeVIPs() error {
	inst.log.Info("[%s] removing virtual IPs", inst.Name)

	for _, vipStr := range inst.getVIPsWithCIDR() {
		has, _ := inst.netInterface.HasIP(vipStr)
		if !has {
			continue
		}

		if err := inst.netInterface.DeleteIP(vipStr); err != nil {
			inst.log.Error("[%s] failed to remove VIP %s: %v", inst.Name, vipStr, err)
			return err
		}
		inst.log.Info("[%s] removed VIP %s", inst.Name, vipStr)
	}

	return nil
}

func (inst *Instance) getVIPsWithCIDR() []string {
	result := make([]string, len(inst.VirtualIPs))
	for i, ip := range inst.VirtualIPs {
		result[i] = ip.String() + "/32"
	}
	return result
}

func (inst *Instance) GetState() State {
	return inst.state.GetState()
}

func (inst *Instance) OnMaster(callback func()) {
	inst.onMaster = callback
}

func (inst *Instance) OnBackup(callback func()) {
	inst.onBackup = callback
}

func (inst *Instance) OnFault(callback func()) {
	inst.onFault = callback
}

func (inst *Instance) AdjustPriority(delta int) {
	inst.mu.Lock()
	defer inst.mu.Unlock()

	oldPriority := inst.priorityCalc.GetPriority()
	
	if delta < 0 {
		inst.priorityCalc.DecreasePriority(uint8(-delta))
	}
	
	newPriority := inst.priorityCalc.GetPriority()
	
	if oldPriority != newPriority {
		inst.log.Info("[%s] priority adjusted: %d -> %d (delta=%d)", 
			inst.Name, oldPriority, newPriority, delta)
	}
}

func (inst *Instance) ResetPriority() {
	inst.mu.Lock()
	defer inst.mu.Unlock()

	oldPriority := inst.priorityCalc.GetPriority()
	inst.priorityCalc.ResetPriority()
	newPriority := inst.priorityCalc.GetPriority()
	
	if oldPriority != newPriority {
		inst.log.Info("[%s] priority reset: %d -> %d", 
			inst.Name, oldPriority, newPriority)
	}
}
