package common

import (
	"bufio"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strings"
)

type sliceValue []string

func newSliceValue(vals []string, p *[]string) *sliceValue {
	*p = vals
	return (*sliceValue)(p)
}

func (s *sliceValue) Set(val string) error {
	*s = sliceValue(strings.Split(val, ","))
	return nil
}

func (s *sliceValue) Get() interface{} {
	return []string(*s)
}

func (s *sliceValue) String() string {
	return strings.Join([]string(*s), ",")
}

var (
	cmdIps []string
	// cmdExPath  string
	cmdCofPath string
	cmdPorts   []string
	cmdT1000   bool
	cmdRandom  bool
	NumThreads int
	excPorts   []string // 待排除端口
	excIps     []string // 待排除Ip
	ipFile     string
	nocolor    bool //彩色打印
	json       bool
	tracelog   string //请求日志
	rstfile    string //文件保存
	verify     bool //验证模式
	skiphttp  bool //不请求httptitle
)

/**
  命令行参数解析：
  -i: 输入的Ip地址或者域名,以逗号分隔. 例如192.168.1.1/24,scanme.nmap.org
  -e: 设置排除文件路径，排除文件内容为需要排除的ip地址列表
  -c: 配置文件路径，支持从配置文件中读取ip，地址列表
  -p: 需要扫描的端口列表，以逗号分隔，例如: 1-1000,3379,6379，和-p互斥
  -t100: 布尔类型，默认是扫描top100，否则扫描top1000端口，和-p互斥
  -r: 布尔类型，表示扫描方式，随机扫描还是顺序扫描
  -v: 验证模式，是否从文件导入进行验证
*/
func init() {
	flag.Var(newSliceValue([]string{}, &cmdIps), "i", "set domain and ips")
	//flag.StringVar(&cmdExPath, "e", "", "set exclude file path")
	// flag.StringVar(&cmdCofPath, "c", "", "set config file path")
	flag.Var(newSliceValue([]string{}, &cmdPorts), "p", "set port ranges to scan，default is top100")

	flag.BoolVar(&cmdT1000, "t1000", false, "scan top1000 ports")
	flag.BoolVar(&cmdRandom, "r", false, "random scan flag")
	flag.IntVar(&NumThreads, "n", 800, "number of goroutines, between 1 and 2000")
	flag.Var(newSliceValue([]string{}, &excPorts), "ep", "set port ranges to exclude")
	flag.Var(newSliceValue([]string{}, &excIps), "ei", "set ip ranges to exclude")
	flag.StringVar(&ipFile, "l", "", "input ips file")
	flag.BoolVar(&nocolor, "nocolor", false, "using color ascii to screen")
	flag.BoolVar(&json, "json", false, "output json format")
	flag.StringVar(&tracelog, "tracefile", "", "request log")
	flag.StringVar(&rstfile, "o", "rst.txt", "success log")
	flag.BoolVar(&verify, "v", false, "verified from file")
	flag.BoolVar(&skiphttp, "sk", false, "skip http request")
}

type Identification_Packet struct {
	Desc   string
	Packet []byte
}

var st_Identification_Packet [100]Identification_Packet

// 初始化IdentificationProtocol到内存中

func init() {
	for i, packet := range IdentificationProtocol {
		szinfo := strings.Split(packet, "#")
		data, err := hex.DecodeString(szinfo[1])
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		st_Identification_Packet[i].Desc = szinfo[0]
		st_Identification_Packet[i].Packet = data
	}
}

func ArgsPrint() {
	fmt.Println(cmdIps)
	fmt.Println(cmdRandom)
	fmt.Println(cmdPorts)
	fmt.Println(excPorts)
}

/**
  configeFileParse 配置文件解析函数
  配置文件每行一条数据，可以是单个ip，域名，也可以是带掩码的ip和域名
*/
func ConfigeFileParse(path string) ([]string, error) {
	var err error
	var ips = make([]string, 0, 100)

	file, err := os.Open(path)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	defer file.Close()

	reader := bufio.NewReader(file)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		// 去除空行
		if len(line) == 0 || line == "\r\n" {
			continue
		}

		// 以#开头的为注释内容
		if strings.Index(line, "#") == 0 {
			continue
		}

		ips = append(ips, line)
	}

	return ips, err
}
