package core

import (
	"bytes"
	"encoding/json"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/spray/core/baseline"
	"github.com/chainreactors/spray/pkg"
	"github.com/chainreactors/words/mask"
	"io"
	"net/url"
	"os"
	"strings"
)

func Format(opts Option) {
	var content []byte
	var err error
	if opts.Format == "stdin" {
		content, err = io.ReadAll(os.Stdin)
	} else {
		content, err = os.ReadFile(opts.Format)
	}

	if err != nil {
		return
	}
	group := make(map[string]map[string]*baseline.Baseline)
	for _, line := range bytes.Split(bytes.TrimSpace(content), []byte("\n")) {
		var result baseline.Baseline
		err := json.Unmarshal(line, &result)
		if err != nil {
			logs.Log.Error(err.Error())
			return
		}
		result.Url, err = url.Parse(result.UrlString)
		if err != nil {
			continue
		}
		if _, exists := group[result.Url.Host]; !exists {
			group[result.Url.Host] = make(map[string]*baseline.Baseline)
		}
		group[result.Url.Host][result.Path] = &result
	}

	for _, results := range group {
		for _, result := range results {
			if !opts.Fuzzy && result.IsFuzzy {
				continue
			}
			if opts.OutputProbe == "" {
				if !opts.NoColor {
					logs.Log.Console(result.ColorString() + "\n")
				} else {
					logs.Log.Console(result.String() + "\n")
				}
			} else {
				probes := strings.Split(opts.OutputProbe, ",")
				logs.Log.Console(result.ProbeOutput(probes) + "\n")
			}
		}
	}
}

// PrintPreset
//
//	@Description: 打印内置配置信息
func PrintPreset() {
	logs.Log.Console("internal rules:\n")
	for name, rule := range pkg.Rules {
		logs.Log.Consolef("\t%s\t%d rules\n", name, len(strings.Split(rule, "\n")))
	}

	logs.Log.Console("\ninternal dicts:\n")
	for name, dict := range pkg.Dicts {
		logs.Log.Consolef("\t%s\t%d items\n", name, len(dict))
	}

	logs.Log.Console("\ninternal words keyword:\n")
	for name, words := range mask.SpecialWords {
		logs.Log.Consolef("\t%s\t%d words\n", name, len(words))
	}

	logs.Log.Console("\ninternal extractor:\n")
	for name, _ := range pkg.ExtractRegexps {
		logs.Log.Consolef("\t%s\n", name)
	}

	logs.Log.Console("\ninternal fingers:\n")
	for name, engine := range pkg.FingerEngine.EnginesImpl {
		logs.Log.Consolef("\t%s\t%d fingerprints \n", name, engine.Len())
	}

	logs.Log.Consolef("\nload %d active path\n", len(pkg.ActivePath))
}
