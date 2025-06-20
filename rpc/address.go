package rpc

import (
	"fmt"
	"math/rand"
	"net"
	"sort"
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

func GetSelfAddress(c *config.Config) *config.Address {
	if c.SelfRpcAddress != nil {
		return c.SelfRpcAddress
	}

	for _, address := range c.RpcAddress {
		if address.Ip == util.GetLocalIP() {
			return address
		}
	}

	return nil
}

func NewAddressManager(c *config.Config) (*AddressManager, error) {
	port, err := getAvailablePortWithLimit()
	if err != nil {
		return nil, err
	}

	localManager := &AddressManager{
		selfMachineId: LOCAL_NAME,
		machineInfo: map[string]*config.Address{
			LOCAL_NAME: {
				Ip:   "127.0.0.1",
				Port: port,
			},
		},
	}

	if len(c.RpcAddress) == 0 {
		return localManager, nil
	}

	selfAddress := GetSelfAddress(c)
	if selfAddress == nil {
		return nil, fmt.Errorf("multi-instance deploy failed, IP %s not found in instances list of configuration", util.GetLocalIP())
	}

	selfId := ""

	rm := map[string]*config.Address{}
	var a []string
	for _, info := range c.RpcAddress {
		key := fmt.Sprintf("%s:%s", info.Ip, info.Port)
		rm[key] = info
		a = append(a, key)
	}

	sort.Strings(a)
	nAddress := map[string]*config.Address{}
	for i, key := range a {
		newKey := fmt.Sprintf("A%d", i)
		address := rm[key]
		nAddress[newKey] = address
		log.Infof("generate local machine ID %s => %s:%s", newKey, address.Ip, address.Port)
		if address.Ip == selfAddress.Ip && address.Port == selfAddress.Port {
			selfId = newKey
		}
	}

	if selfId == "" {
		return nil, fmt.Errorf("multi-instance deploy failed, generate local machine ID failed")
	}

	log.Infof("current instance ID %s", selfId)
	m := &AddressManager{
		selfMachineId: selfId,
		machineInfo:   nAddress,
	}

	return m, nil
}

type AddressManager struct {
	selfMachineId string
	machineInfo   map[string]*config.Address
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
	return a.selfMachineId
}

func (a *AddressManager) IsSelfMachineAddress(address *event.Address) bool {
	return a.GetSelfMachineID() == address.MachineID
}

func (a *AddressManager) GetSelfAddress() *config.Address {
	return a.GetMachineIpInfo()[a.GetSelfMachineID()]
}

func (a *AddressManager) GetMachineIpInfo() map[string]*config.Address {
	return a.machineInfo
}
