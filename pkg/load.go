package pkg

import (
	"fmt"
	"os"
	"strings"

	"github.com/chainreactors/fingers"
	"github.com/chainreactors/parsers"
	"github.com/chainreactors/utils"
	"github.com/chainreactors/utils/iutils"
	"github.com/chainreactors/words/mask"
	yaml "sigs.k8s.io/yaml/goyaml.v3"
)

func LoadPorts() error {
	var err error
	var ports []*utils.PortConfig
	err = yaml.Unmarshal(LoadConfig("port"), &ports)
	if err != nil {
		return err
	}
	utils.PrePort = utils.NewPortPreset(ports)
	return nil
}

func LoadFingers() error {
	var err error
	FingerEngine, err = fingers.NewEngine()
	if err != nil {
		return err
	}
	for _, f := range FingerEngine.Fingers().HTTPFingers {
		for _, rule := range f.Rules {
			if rule.SendDataStr != "" {
				ActivePath = append(ActivePath, rule.SendDataStr)
			}
		}
	}
	for _, f := range FingerEngine.FingerPrintHub().FingerPrints {
		if f.Path != "/" {
			ActivePath = append(ActivePath, f.Path)
		}
	}
	return nil
}

func LoadTemplates() error {
	var err error
	// load rule

	err = yaml.Unmarshal(LoadConfig("spray_rule"), &Rules)
	if err != nil {
		return err
	}

	// load default words
	var dicts map[string]string
	err = yaml.Unmarshal(LoadConfig("spray_dict"), &dicts)
	if err != nil {
		return err
	}
	for name, wordlist := range dicts {
		dict := strings.Split(strings.TrimSpace(wordlist), "\n")
		for i, d := range dict {
			dict[i] = strings.TrimSpace(d)
		}
		Dicts[strings.TrimSuffix(name, ".txt")] = dict
	}

	// load mask
	var keywords map[string]interface{}
	err = yaml.Unmarshal(LoadConfig("spray_common"), &keywords)
	if err != nil {
		return err
	}

	for k, v := range keywords {
		t := make([]string, len(v.([]interface{})))
		for i, vv := range v.([]interface{}) {
			t[i] = iutils.ToString(vv)
		}
		mask.SpecialWords[k] = t
	}

	var extracts []*parsers.Extractor
	err = yaml.Unmarshal(LoadConfig("extract"), &extracts)
	if err != nil {
		return err
	}

	for _, extract := range extracts {
		extract.Compile()

		ExtractRegexps[extract.Name] = []*parsers.Extractor{extract}
		for _, tag := range extract.Tags {
			if _, ok := ExtractRegexps[tag]; !ok {
				ExtractRegexps[tag] = []*parsers.Extractor{extract}
			} else {
				ExtractRegexps[tag] = append(ExtractRegexps[tag], extract)
			}
		}
	}
	return nil
}

func LoadExtractorConfig(filename string) ([]*parsers.Extractor, error) {
	var extracts []*parsers.Extractor
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(content, &extracts)
	if err != nil {
		return nil, err
	}

	for _, extract := range extracts {
		extract.Compile()
	}

	return extracts, nil
}

// Load
//
//	@Description: 加载默认端口和模板数据,模板数据包含:rules,dicts,words keyword,extractor,fingers
//	@return error
func Load() error {
	err := LoadPorts()
	if err != nil {
		return fmt.Errorf("load ports, %w", err)
	}
	err = LoadTemplates()
	if err != nil {
		return fmt.Errorf("load templates, %w", err)
	}

	return nil
}
