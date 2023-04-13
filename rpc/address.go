package rpc

import (
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"time"

	"github.com/HuolalaTech/page-spy-api/api/event"
	"github.com/HuolalaTech/page-spy-api/config"
	"github.com/HuolalaTech/page-spy-api/util"
	"github.com/google/uuid"
)

const LOCAL_NAME = "local"

func newAddressID(localID string, machineID string) string {
	return fmt.Sprintf("%s.%s", localID, machineID)
}

func getAvailablePort(limit int) (string, error) {
	if limit <= 0 {
		return "", fmt.Errorf("get available port try times than limit")
	}
	rand.Seed(time.Now().Unix())
	min := 1024
	max := 65535
	port := rand.Intn(max-min) + min

	fmt.Println(port)
	// 检查端口号是否可用
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		// 端口号已被占用，需要重新获取
		return getAvailablePort(limit - 1)
	}
	defer listener.Close()

	return strconv.Itoa(port), nil
}

func getAvailablePortWithLimit() (string, error) {
	return getAvailablePort(5)
}

func NewAddressManager(c *config.Config) (*AddressManager, error) {
	port, err := getAvailablePortWithLimit()
	if err != nil {
		return nil, err
	}

	localManager := &AddressManager{
		machineInfo: &config.MachineInfo{
			MachineAddress: map[string]*config.Address{
				LOCAL_NAME: {
					Ip:   "127.0.0.1",
					Port: port,
				},
			},
		},
	}

	if c.MachineInfo == nil || len(c.MachineInfo.MachineAddress) <= 0 {
		return localManager, nil
	}

	m := &AddressManager{
		machineInfo: c.MachineInfo,
	}

	// 如果没有注册机器则开启本地模式
	if m.GetSelfMachineID() == LOCAL_NAME {
		return localManager, nil
	}

	return m, nil
}

type AddressManager struct {
	machineInfo *config.MachineInfo
}

func (a *AddressManager) GeneratorConnectionAddress() *event.Address {
	mID := a.GetSelfMachineID()
	lID := a.GeneratorLocalID()
	return &event.Address{
		ID:        newAddressID(lID, mID),
		MachineID: mID,
		LocalID:   lID,
	}
}

func (a *AddressManager) GeneratorRoomAddress() *event.Address {
	mID := a.GetSelfMachineID()
	lID := a.GeneratorLocalID()
	return &event.Address{
		ID:        newAddressID(lID, mID),
		MachineID: mID,
		LocalID:   lID,
	}
}

func (a *AddressManager) GeneratorLocalID() string {
	return uuid.New().String()
}

func (a *AddressManager) GetSelfMachineID() string {
	for id, info := range a.machineInfo.MachineAddress {
		if info.Ip == util.GetLocalIP() {
			return id
		}
	}

	return LOCAL_NAME
}

func (a *AddressManager) IsSelfMachineAddress(address *event.Address) bool {
	return a.GetSelfMachineID() == address.MachineID
}

func (a *AddressManager) GetSelfAddress() *config.Address {
	return a.GetMachineIpInfo()[a.GetSelfMachineID()]
}

func (a *AddressManager) GetMachineIpInfo() map[string]*config.Address {
	return a.machineInfo.MachineAddress
}
