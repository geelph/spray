// 程序命令行方式入口文件
package cmd

// 引入标准库和三方库
import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chainreactors/files"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/spray/core"
	"github.com/chainreactors/spray/core/ihttp"
	"github.com/chainreactors/spray/pkg"
	"github.com/chainreactors/utils/iutils"
	"github.com/jessevdk/go-flags"
)

var ver = "dev"
var DefaultConfig = "config.yaml"

// init
//
//	@Description: 包初始化函数，配置日志库颜色输出
func init() {
	logs.Log.SetColorMap(map[logs.Level]func(string) string{
		logs.Info:      logs.PurpleBold,
		logs.Important: logs.GreenBold,
		pkg.LogVerbose: logs.Green,
	})
}

// Spray
//
//	@Description: 程序核心入口函数
func Spray() {
	// 声明参数结构变量
	var option core.Option

	// 设置日志等级
	logs.AddLevel(pkg.LogVerbose, "verbose", "[=] %s {{suffix}}\n")
	logs.Log.SetLevel(logs.Debug)

	// 判断默认配置文件是否存在，并尝试加载配置文件
	if files.IsExist(DefaultConfig) {
		// 打印日志
		logs.Log.Debug("config.yaml exist, loading")
		// 加载配置文件
		err := core.LoadConfig(DefaultConfig, &option)
		// 加载配置文件发生错误
		if err != nil {
			logs.Log.Error(err.Error())
			return
		}
	}

	// 命令行参数设计
	parser := flags.NewParser(&option, flags.Default)
	// 用法说明
	parser.Usage = `

  WIKI: https://chainreactors.github.io/wiki/spray
  
  QUICKSTART:
	basic:
	  spray -u http://example.com

	basic cidr and port:
	  spray -i example -p top2,top3

    simple brute:
      spray -u http://example.com -d wordlist1.txt -d wordlist2.txt

    mask-base brute with wordlist:
      spray -u http://example.com -w "/aaa/bbb{?l#4}/ccc"

    rule-base brute with wordlist:
      spray -u http://example.com -r rule.txt -d 1.txt

    list input spray:
      spray -l url.txt -r rule.txt -d 1.txt

    resume:
      spray --resume stat.json
`

	// 参数解析,错误结束
	_, err := parser.Parse()
	if err != nil {
		if err.(*flags.Error).Type != flags.ErrHelp {
			fmt.Println(err.Error())
		}
		return
	}

	// logs
	//logs.AddLevel(pkg.LogVerbose, "verbose", "[=] %s {{suffix}}\n")
	// 更具选项设置日志等级
	if option.Debug {
		logs.Log.SetLevel(logs.Debug)
	} else if len(option.Verbose) > 0 {
		logs.Log.SetLevel(pkg.LogVerbose)
	}
	// 配置文件初始化选项判断
	if option.InitConfig {
		// 从参数选项动态生成配置字符串
		configStr := core.InitDefaultConfig(&option, 0)
		// 判断文件是否已存在,存在则覆盖
		if files.IsExist(DefaultConfig) {
			logs.Log.Warn("override default config: ./config.yaml")
		}
		// 创建默认配置文件
		err := os.WriteFile(DefaultConfig, []byte(configStr), 0o744)
		// 创建失败
		if err != nil {
			logs.Log.Warn("cannot create config: config.yaml, " + err.Error())
			return
		}
		// 初始化配置文件成功
		logs.Log.Info("init default config: ./config.yaml")
		return
	}

	// 在当前函数执行完毕并准备返回后，让程序暂停 1 秒钟
	defer time.Sleep(time.Second)
	// 若指定配置文件参数则尝试加载
	if option.Config != "" {
		// 尝试加载config参数
		err := core.LoadConfig(option.Config, &option)
		// 加载失败处理
		if err != nil {
			logs.Log.Error(err.Error())
			return
		}
		// 默认配置文件存在,提示
		if files.IsExist(DefaultConfig) {
			logs.Log.Warnf("custom config %s, override default config", option.Config)
		} else {
			logs.Log.Important("load config: " + option.Config)
		}
	}

	// 打印版本信息
	if option.Version {
		fmt.Println(ver)
		return
	}

	// 打印内置配置
	if option.PrintPreset {
		// 加载默认端口和模板数据
		err = pkg.Load()
		// 加载失败
		if err != nil {
			iutils.Fatal(err.Error())
		}

		// 加载指纹
		err = pkg.LoadFingers()
		// 加载失败时处理
		if err != nil {
			iutils.Fatal(err.Error())
		}
		// 调用函数打印内置配置信息
		core.PrintPreset()

		return
	}

	// 判断格式参数
	if option.Format != "" {
		core.Format(option)
		return
	}

	err = option.Prepare()
	if err != nil {
		logs.Log.Errorf(err.Error())
		return
	}

	runner, err := option.NewRunner()
	if err != nil {
		logs.Log.Errorf(err.Error())
		return
	}
	if option.ReadAll || runner.CrawlPlugin {
		ihttp.DefaultMaxBodySize = -1
	}

	ctx, canceler := context.WithTimeout(context.Background(), time.Duration(runner.Deadline)*time.Second)
	go func() {
		select {
		case <-ctx.Done():
			time.Sleep(10 * time.Second)
			logs.Log.Errorf("deadline and timeout not work, hard exit!!!")
			os.Exit(0)
		}
	}()

	go func() {
		exitChan := make(chan os.Signal, 2)
		signal.Notify(exitChan, os.Interrupt, syscall.SIGTERM)

		go func() {
			sigCount := 0
			for {
				<-exitChan
				sigCount++
				if sigCount == 1 {
					logs.Log.Infof("Exit signal received, saving task and exiting...")
					canceler()
				} else if sigCount == 2 {
					logs.Log.Infof("forcing exit...")
					os.Exit(1)
				}
			}
		}()
	}()

	err = runner.Prepare(ctx)
	if err != nil {
		logs.Log.Errorf(err.Error())
		return
	}

}
