//go:generate go run templates/templates_gen.go -t templates -o pkg/templates.go -need spray
package main

// 引入本地包和三方包
import (
	"github.com/chainreactors/spray/cmd"
	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
	//_ "net/http/pprof"
)

// init
//
//	@Description: 配置初始化
func init() {
	// 添加额外选项功能
	config.WithOptions(func(opt *config.Options) {
		// 更改结构标签名称
		opt.DecoderConfig.TagName = "config"
		// 解析默认值
		opt.ParseDefault = true
	})
	// 添加yaml格式驱动器
	config.AddDriver(yaml.Driver)
}

// main
//
//	@Description: 程序主函数
func main() {
	//f, _ := os.Create("cpu.txt")
	//pprof.StartCPUProfile(f)
	//defer pprof.StopCPUProfile()
	// 调用cmd的函数
	cmd.Spray()
}
