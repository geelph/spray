package core

import (
	"fmt"
	"github.com/chainreactors/files"
	"github.com/chainreactors/fingers"
	"github.com/chainreactors/fingers/resources"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/utils/encode"
	"github.com/chainreactors/utils/iutils"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var (
	// 默认配置文件路径
	DefaultFingerPath     = "fingers"
	DefaultFingerTemplate = "fingers/templates"

	// 指纹配置文件名称
	FingerConfigs = map[string]string{
		fingers.FingersEngine:     "fingers_http.json.gz",
		fingers.FingerPrintEngine: "fingerprinthub_v3.json.gz",
		fingers.WappalyzerEngine:  "wappalyzer.json.gz",
		fingers.EHoleEngine:       "ehole.json.gz",
		fingers.GobyEngine:        "goby.json.gz",
	}
	// 配置文件资源文件URL,下载指纹配置时用到
	baseURL = "https://raw.githubusercontent.com/chainreactors/fingers/master/resources/"
)

// FingerOptions
// @Description: 定义指纹类型
type FingerOptions struct {
	// 启动指纹检测
	Finger bool `long:"finger" description:"Bool, enable active finger detect" config:"finger"`
	// 指纹更新
	FingerUpdate bool `long:"update" description:"Bool, update finger database" config:"update"`
	// 指纹路径
	FingerPath string `long:"finger-path" default:"fingers" description:"String, 3rd finger config path" config:"finger-path"`
	//FingersTemplatesPath string `long:"finger-template" default:"fingers/templates" description:"Bool, use finger templates path" config:"finger-template"`
	// 指纹引擎
	FingerEngines string `long:"finger-engine" default:"all" description:"String, custom finger engine, e.g. --finger-engine ehole,goby" config:"finger-engine"`
}

// Validate
//
//	@Description: 验证指纹相关参数是否正确;指纹更新参数是否指定,并生成对应指纹路径(自定义或默认);判断指纹引擎是否正确
//	@receiver opt 指纹参数类型
//	@return error 返回错误类型
func (opt *FingerOptions) Validate() error {
	// 声明error类型变量
	var err error
	// 判断指纹更新参数是否存在
	if opt.FingerUpdate {
		// 当纹路径参数不等于默认指纹路径,且执行指纹路径参数不存在时
		if opt.FingerPath != DefaultFingerPath && !files.IsExist(opt.FingerPath) {
			// 创建指纹路径并赋予权限
			err = os.MkdirAll(opt.FingerPath, 0755)
			// 判断是否创建成功
			if err != nil {
				return err
			}
		} else if !files.IsExist(DefaultFingerPath) { // 当默认路径不存在
			// 将默认指纹路径赋于指纹路径参数
			opt.FingerPath = DefaultFingerPath
			// 创建默认指纹路径并赋于权限
			err = os.MkdirAll(DefaultFingerPath, 0755)
			if err != nil {
				return err
			}
		}
		//if opt.FingersTemplatesPath != DefaultFingerTemplate && !files.IsExist(opt.FingersTemplatesPath) {
		//	err = os.MkdirAll(opt.FingersTemplatesPath, 0755)
		//	if err != nil {
		//		return err
		//	}
		//} else if !files.IsExist(DefaultFingerTemplate) {
		//	err = os.MkdirAll(DefaultFingerTemplate, 0755)
		//	if err != nil {
		//		return err
		//	}
		//}
	}

	// 当指纹引擎参数不是all时
	if opt.FingerEngines != "all" {
		// 对指纹引擎参数按照逗号进行分割遍历
		for _, name := range strings.Split(opt.FingerEngines, ",") {
			// 当指定指纹引擎不存在时,打印错误信息
			// 不区分大小写的引擎名称
			if !iutils.StringsContains(fingers.AllEngines, name) {
				return fmt.Errorf("invalid finger engine: %s, please input one of %v", name, fingers.FingersEngine)
			}
		}
	}
	return nil
}

// LoadLocalFingerConfig
//
//	@Description: 记载本地指纹配置
//	@receiver opt 指纹参数类型
//	@return error 返回错误类型
func (opt *FingerOptions) LoadLocalFingerConfig() error {
	// 遍历指纹引擎名称及指纹配置文件名
	for name, fingerPath := range FingerConfigs {
		// 拼接正确文件路径
		// 直接使用 opt.FingerPath 是以为 Validate() 中会修正目录名称，要不是自定义要不是默认
		fingerPath = opt.FingerPath + "/" + fingerPath
		//fingerPath = filepath.Join(files.GetExcPath(), opt.FingerPath, fingerPath)
		// 尝试读取指纹配置路径
		if content, err := os.ReadFile(fingerPath); err == nil {
			// 根据MD5判断文件是否发生变化,不一致时使用本地替换
			if encode.Md5Hash(content) != resources.CheckSum[name] {
				// 发现不同时,使用本地
				logs.Log.Importantf("found %s difference, use %s replace embed", name, fingerPath)
				// 根据引擎名称
				switch name {
				case fingers.FingersEngine:
					resources.FingersHTTPData = content
				case fingers.FingerPrintEngine:
					resources.Fingerprinthubdata = content
				case fingers.EHoleEngine:
					resources.EholeData = content
				case fingers.GobyEngine:
					resources.GobyData = content
				case fingers.WappalyzerEngine:
					resources.WappalyzerData = content
				default:
					return fmt.Errorf("unknown engine name: %s", name)
				}
				logs.Log.Debugf("%s config is loading", name)
			} else {
				// 配置是最新的
				logs.Log.Infof("%s config is up to date", name)
			}
		} else {
			// 读取指纹配置文件路径时发生错误
			logs.Log.Errorf("An error occurred when reading the %s fingerprint configuration file:%s", name, err)
		}
	}
	return nil
}

// UpdateFinger
//
//	@Description: 升级指纹配置
//	@receiver opt 指纹参数类型
//	@return error 返回错误类型
func (opt *FingerOptions) UpdateFinger() error {
	// 修改标识
	modified := false
	// 遍历指纹配置
	for name, _ := range FingerConfigs {
		// 下载指纹配置文件
		if ok, err := opt.downloadConfig(name); err != nil {
			return err
		} else {
			if ok {
				modified = ok
			}
		}
	}
	// 未被修改,本地是最新的
	if !modified {
		logs.Log.Importantf("everything is up to date")
	}
	return nil
}

// downloadConfig
//
//	@Description: 下载指纹配置
//	@receiver opt 指纹配置类型
//	@param name 指纹引擎名称
//	@return bool 返回方法是否执行成功
//	@return error 返回下载操作是否执行成功
func (opt *FingerOptions) downloadConfig(name string) (bool, error) {
	// 获取执行配置文件名
	fingerFile, ok := FingerConfigs[name]
	// 指定引擎参数错误
	if !ok {
		return false, fmt.Errorf("unknown engine name")
	}
	// 拼接指纹配置资源下载路径
	url := baseURL + fingerFile
	// 使用使用http方式下载指纹配置文件资源
	resp, err := http.Get(url)
	// 下载失败处理
	if err != nil {
		return false, err
	}
	// 延迟,本函数即将执行完毕后关闭这次请求连接
	defer resp.Body.Close()

	// 根据状态码判断是否下载成功
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("bad status: %s", resp.Status)
	}

	// 读取响应体内容
	content, err := io.ReadAll(resp.Body)
	// 读取内容失败时返回
	if err != nil {
		return false, err
	}
	// 拼接指纹配置文件路径
	filePath := filepath.Join(files.GetExcPath(), opt.FingerPath, fingerFile)
	// 当路径存在时
	if files.IsExist(filePath) {
		// 读取存在的文件
		origin, err := os.ReadFile(filePath)
		if err != nil {
			return false, err
		}
		// 当两个文件内容的MD5值不一致时,覆盖
		if resources.CheckSum[name] != encode.Md5Hash(origin) {
			logs.Log.Importantf("update %s config from %s save to %s", name, url, fingerFile)
			// 写到本地
			err = os.WriteFile(filePath, content, 0644)
			if err != nil {
				return false, err
			}
			return true, nil
		}
	} else {
		// 当文件不存在时,进行创建并写入本地
		out, err := os.Create(filePath)
		if err != nil {
			return false, err
		}
		defer out.Close()
		logs.Log.Importantf("download %s config from %s save to %s", name, url, fingerFile)
		err = os.WriteFile(filePath, content, 0644)
		if err != nil {
			return false, err
		}
	}

	// 读文件并做Hash比较
	if origin, err := os.ReadFile(filePath); err == nil {
		// 与云端不同时,进行覆盖
		if encode.Md5Hash(content) != encode.Md5Hash(origin) {
			logs.Log.Infof("download %s config from %s save to %s", name, url, fingerFile)
			// 覆盖本地文件
			err = os.WriteFile(filePath, content, 0644)
			if err != nil {
				return false, err
			}
			return true, nil
		}
	}

	return false, nil
}
