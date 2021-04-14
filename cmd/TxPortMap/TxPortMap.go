package main

import (
	"flag"
	"fmt"
	 _ "github.com/projectdiscovery/fdmax/autofdmax"	//Add automatic increase of file descriptors in linux
	"github.com/4dogs-cn/TXPortMap/pkg/common"
	"os"
)

func init() {
	flag.Parse()

	// fmt.Println("threadnum: ", common.NumThreads)

	if common.NumThreads < 1 || common.NumThreads > 2000 {
		fmt.Println("number of goroutine must between 1 and 2000")
		os.Exit(-1)
	}
}

// 建议扫描top100或者top1000端口时使用顺序扫描，其它情况使用随机扫描
func main() {

	// trace追踪文件生产，调试时打开注释即可
	/*
		f1, err := os.Create("scan.trace")
		if err != nil {
			log.Fatal(err)
		}
		trace.Start(f1)
		defer trace.Stop()
	*/
	//common.ArgsPrint()
	engine := common.CreateEngine()

	// 命令行参数错误
	if err := engine.Parser(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// common.ArgsPrint()
	//engine.Wg.Add(engine.WorkerCount)
	//go engine.Scheduler()
	engine.Run()

	// 等待扫描任务完成
	engine.Wg.Wait()
	if common.Writer !=nil{
		common.Writer.Close()
	}
}
