package pkg

import (
	"bytes"
	"github.com/chainreactors/fingers/common"
)

// gogo fingers engine
func FingersDetect(content []byte) common.Frameworks {
	// 规则引擎匹配前将待匹配的内容转换为小写
	frames, _ := FingerEngine.Fingers().HTTPMatch(bytes.ToLower(content), "")
	return frames
}

func EngineDetect(content []byte) common.Frameworks {
	// 尝试转小写
	frames, _ := FingerEngine.DetectContent(content)
	return frames
}
