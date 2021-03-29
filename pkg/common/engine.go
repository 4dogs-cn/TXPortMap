package common

import (
	"fmt"
	"github.com/4dogs-cn/TXPortMap/pkg/Ginfo/Ghttp"
	ps "github.com/4dogs-cn/TXPortMap/pkg/common/ipparser"
	rc "github.com/4dogs-cn/TXPortMap/pkg/common/rangectl"
	"github.com/4dogs-cn/TXPortMap/pkg/conversion"
	"github.com/4dogs-cn/TXPortMap/pkg/output"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

type Addr struct {
	ip   string
	port uint64
}

// type Range struct {
// 	Begin uint64
// 	End   uint64
// }

var (
	Writer output.Writer
)

type Engine struct {
	TaskIps     []rc.Range
	TaskPorts   []rc.Range
	ExcdPorts   []rc.Range // 待排除端口
	ExcdIps     []rc.Range // 待排除的Ip
	RandomFlag  bool
	WorkerCount int
	TaskChan    chan Addr // 传递待扫描的ip端口对
	//DoneChan chan struct{}  // 任务完成通知
	Wg *sync.WaitGroup
}

// 扫描目标建立，ip:port发送到任务通道
func (e *Engine) Run() {
	var addr Addr

	e.Wg.Add(e.WorkerCount)
	go e.Scheduler()

	// fmt.Println(e.TaskPorts)

	// TODO:: if !e.RandomFlag
	if !e.RandomFlag {
		// 随机扫描，向任务通道随机发送addr
		e.randomScan()

	} else {
		// 顺序扫描，向任务通道顺序发送addr
		for _, ipnum := range e.TaskIps {
			for ips := ipnum.Begin; ips <= ipnum.End; ips++ {
				ip := ps.UnParseIPv4(ips)

				for _, ports := range e.TaskPorts {
					for port := ports.Begin; port <= ports.End; port++ {
						addr.ip = ip
						addr.port = port

						//e.SubmitTask(addr)
						//fmt.Println("ip:",ip,":port",port)
						e.TaskChan <- addr
					}
				}
			}
		}
	}

	// 扫描任务发送完成，关闭通道
	//fmt.Println("Task Add done")
	close(e.TaskChan)
}

func (e *Engine) SubmitTask(addr Addr) {
	//fmt.Printf("submit# %s:%d\n", addr.ip, addr.port)
	go func() {
		e.TaskChan <- addr
	}()
}

// 扫描任务创建
func (e *Engine) Scheduler() {
	for i := 0; i < e.WorkerCount; i++ {
		worker(e.TaskChan, e.Wg)
	}
}

// 参数解析，对命令行中传递的参数进行格式化存储
func (e *Engine) Parser() error {
	var err error
	Writer, err = output.NewStandardWriter(nocolor,false, rstfile, tracelog)
	if err != nil {
		return err
	}
	var ports []string
	// TODO:: 待增加排除ip和排除端口流程

	for _, ipstr := range cmdIps {
		if ps.IsIP(ipstr) || ps.IsIPRange(ipstr) {
			result, err := rc.ParseIpv4Range(ipstr)
			if err != nil {
				fmt.Println("Error occured while parse iprange")
				return err
			}

			e.TaskIps = append(e.TaskIps, result)
		} else {
			// 说明是域名，需要对域名进行解析
			ips, mask, err := ps.DomainToIp(ipstr)
			if err != nil {
				fmt.Println(err)
				return err
			}
			for _, ip := range ips {
				addr := ip
				if mask != "" {
					addr = ip + "/" + mask
				}

				result, err := rc.ParseIpv4Range(addr)

				if err != nil {
					fmt.Println("Error occured while parse iprange")
					return err
				}

				e.TaskIps = append(e.TaskIps, result)
			}
		}
	}

	if ipFile != "" {
		rst, err := rc.ParseIPFromFile(ipFile)
		if err == nil {
			for _, r := range rst {
				e.TaskIps = append(e.TaskIps, r)
			}
		}
	}

	if len(excIps) != 0 {
		for _, ipstr := range excIps {
			if ps.IsIP(ipstr) || ps.IsIPRange(ipstr) {
				result, err := rc.ParseIpv4Range(ipstr)
				if err != nil {
					fmt.Println("Error occured while parse iprange")
					return err
				}

				e.ExcdIps = append(e.ExcdIps, result)
			} else {
				// 说明是域名，需要对域名进行解析
				ips, mask, err := ps.DomainToIp(ipstr)
				if err != nil {
					fmt.Println(err)
					return err
				}
				for _, ip := range ips {
					addr := ip
					if mask != "" {
						addr = ip + "/" + mask
					}

					result, err := rc.ParseIpv4Range(addr)

					if err != nil {
						fmt.Println("Error occured while parse iprange")
						return err
					}

					e.ExcdIps = append(e.ExcdIps, result)
				}
			}
		}

		for _, ipe := range e.ExcdIps {
			for i := 0; i < len(e.TaskIps); i++ {
				if res, ok := (e.TaskIps[i]).RemoveExcFromTaskIps(ipe); ok {
					e.TaskIps = append(e.TaskIps, res)
				}
			}
		}
	}

	// 说明有自定义端口
	if len(cmdPorts) != 0 {
		ports = cmdPorts
	} else {
		if !cmdT1000 {
			// Top100端口扫描
			ports = Top100Ports

		} else {
			// Top1000端口扫描
			ports = Top1000Ports
		}
	}

	// 解析命令行端口范围
	for _, portstr := range ports {
		result, err := rc.ParsePortRange(portstr)
		if err != nil {
			fmt.Println(err)
			return err
		}

		e.TaskPorts = append(e.TaskPorts, result)
	}

	// 解析待排除端口范围
	if len(excPorts) != 0 {
		for _, portstr := range excPorts {
			result, err := rc.ParsePortRange(portstr)
			if err != nil {
				fmt.Println(err)
				return err
			}

			e.ExcdPorts = append(e.ExcdPorts, result)
		}

		// range出来的其实是原始值的拷贝，因此，这里需要对原始值进行修改时，不能使用range
		for _, exp := range e.ExcdPorts {
			for i := 0; i < len(e.TaskPorts); i++ {
				if res, ok := (e.TaskPorts[i]).RemoveExcFromTaskIps(exp); ok {
					e.TaskPorts = append(e.TaskPorts, res)
				}
			}
		}
	}

	// fmt.Println(e.TaskPorts)
	// fmt.Println(e.ExcdPorts)

	return nil
}

func CreateEngine() *Engine {
	return &Engine{
		RandomFlag:  cmdRandom,
		TaskChan:    make(chan Addr, 1000),
		WorkerCount: NumThreads,
		Wg:          &sync.WaitGroup{},
	}
}

func scanner(ip string, port uint64) {
	var dwSvc int
	var iRule = -1
	var bIsIdentification = false
	//var iCntTimeOut = 0

	// 端口开放状态，发送报文，获取响应
	// 先判断端口是不是优先识别协议端口
	for _, svc := range St_Identification_Port {
		if port == svc.Port {
			bIsIdentification = true
			iRule = svc.Identification_RuleId
			data := st_Identification_Packet[iRule].Packet

			dwSvc = SendIdentificationPacketFunction(data, ip, port)

			break
		}
	}

	if dwSvc > UNKNOWN_PORT && dwSvc <= SOCKET_CONNECT_FAILED {
		return
	}

	if dwSvc == SOCKET_READ_TIMEOUT {
		return
	}

	// 发送其他协议查询包
	for i := 0; i < iPacketMask; i++ {
		// 超时2次,不再识别

		if i == 0 {
			// 说明是http，数据需要拼装一下
			var szOption string
			if port == 80 {
				szOption = fmt.Sprintf("%s%s\r\n\r\n", st_Identification_Packet[0].Packet, ip)
			} else {
				szOption = fmt.Sprintf("%s%s:%d\r\n\r\n", st_Identification_Packet[0].Packet, ip, port)
			}

			dwSvc = SendIdentificationPacketFunction([]byte(szOption), ip, port)
			if dwSvc > UNKNOWN_PORT && dwSvc <= SOCKET_CONNECT_FAILED {
				return
			}
		}

		if bIsIdentification && iRule == i {
			continue
		}

		if dwSvc == SOCKET_READ_TIMEOUT {
			return
		}

		dwSvc = SendIdentificationPacketFunction(st_Identification_Packet[i].Packet, ip, port)
		if dwSvc > UNKNOWN_PORT && dwSvc <= SOCKET_CONNECT_FAILED {
			return
		}

		if dwSvc == SOCKET_READ_TIMEOUT {
			return
		}
	}

}

func worker(res chan Addr, wg *sync.WaitGroup) {
	go func() {
		defer wg.Done()

		for addr := range res {
			scanner(addr.ip, addr.port)
		}

	}()
}

func SendIdentificationPacketFunction(data []byte, ip string, port uint64) int {
	addr := fmt.Sprintf("%s:%d", ip, port)
	even := &output.ResultEvent{
		Target: addr,
		Info:   &output.Info{},
	}

	//fmt.Println(addr)
	var dwSvc int = UNKNOWN_PORT

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		// 端口是closed状态
		Writer.Request(ip,conversion.ToString(port),"tcp",fmt.Errorf("time out"))
		return SOCKET_CONNECT_FAILED
	}

	defer conn.Close()

	// Write方法是非阻塞的

	if _, err := conn.Write(data); err != nil {
		Writer.Request(ip, conversion.ToString(port), "tcp", err)
		return dwSvc
	}

	// 直接开辟好空间，避免底层数组频繁申请内存
	var fingerprint = make([]byte, 0, 65535)
	var tmp = make([]byte, 256)
	// 存储读取的字节数
	var num int
	var szBan string
	var szSvcName string

	// 这里设置成6秒是因为超时的时候会重新尝试5次，

	readTimeout := 2 * time.Second

	// 设置读取的超时时间为6s
	conn.SetReadDeadline(time.Now().Add(readTimeout))

	for {
		// Read是阻塞的
		n, err := conn.Read(tmp)
		if err != nil {
			// 虽然数据读取错误，但是端口仍然是open的
			// fmt.Println(err)
			if err != io.EOF {
				dwSvc = SOCKET_READ_TIMEOUT
				// fmt.Printf("Discovered open port\t%d\ton\t%s\n", port, ip)
			}
			break
		}

		if n > 0 {
			num += n
			fingerprint = append(fingerprint, tmp[:n]...)
		} else {
			// 虽然没有读取到数据，但是端口仍然是open的
			// fmt.Printf("Discovered open port\t%d\ton\t%s\n", port, ip)

			break
		}
	}
	Writer.Request(ip,conversion.ToString(port),"tcp",err)
	// 服务识别
	if num > 0 {
		dwSvc = ComparePackets(fingerprint, num, &szBan, &szSvcName)
		if len(szBan) > 15{
			szBan = szBan[:15]
		}
		if dwSvc > UNKNOWN_PORT && dwSvc < SOCKET_CONNECT_FAILED {
			//even.WorkingEvent = "found"
			if szSvcName == "ssl/tls" || szSvcName == "http"{
				 rst := Ghttp.GetHttpTitle(ip,szSvcName,int(port))
				 even.WorkingEvent = rst
			}else{
				even.Info.Banner = strings.TrimSpace(szBan)
				even.Info.Service = szSvcName

			}
			even.Time = time.Now()
			// fmt.Printf("Discovered open port\t%d\ton\t%s\t\t%s\t\t%s\n", port, ip, szSvcName, strings.TrimSpace(szBan))
			Writer.Write(even)
			return dwSvc
		}
	}

	return dwSvc
}

// randomScan 随机扫描, 有问题，扫描C段时扫描不到，
// TODO::尝试遍历ip，端口顺序打乱扫描
func (e *Engine) randomScan() {
	// 投机取巧，打乱端口顺序，遍历ip扫描
	var portlist = make(map[int]uint64)
	var index int
	var addr Addr

	for _, ports := range e.TaskPorts {
		for port := ports.Begin; port <= ports.End; port++ {
			portlist[index] = port
			index++
		}
	}

	for _, ipnum := range e.TaskIps {
		for ips := ipnum.Begin; ips <= ipnum.End; ips++ {
			ip := ps.UnParseIPv4(ips)
			for _, po := range portlist {
				addr.ip = ip
				addr.port = po

				e.TaskChan <- addr
			}
			// fmt.Printf("%d ", po)
		}
	}

}

// 统计待扫描的ip数目
func (e *Engine) ipRangeCount() uint64 {
	var count uint64
	for _, ipnum := range e.TaskIps {
		count += ipnum.End - ipnum.Begin + 1
	}

	return count
}

// 统计待扫描的端口数目
func (e *Engine) portRangeCount() uint64 {
	var count uint64
	for _, ports := range e.TaskPorts {
		count += ports.End - ports.Begin + 1
	}

	return count
}
