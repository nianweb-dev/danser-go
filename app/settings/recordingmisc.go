package settings

import (
	"fmt"
	"strings"
)

type h264V4L2Settings struct {
	RateControl       string `combo:"vbr|VBR,cbr|CBR"`
	Bitrate           string `showif:"RateControl=vbr,cbr"`
	AdditionalOptions string
}

func (s *h264V4L2Settings) GenerateFFmpegArgs() (ret []string, err error) {
	ret, err = v4l2Common(s.RateControl, s.Bitrate)
	if err != nil {
		return nil, err
	}
	return append(ret), nil
}

func v4l2Common(rateControl, bitrate string) (ret []string, err error) {
	switch strings.ToLower(rateControl) {
	case "vbr":
		ret = append(ret, "-b:v", bitrate)
	case "cbr":
		ret = append(ret, "-b:v", bitrate, "-maxrate", bitrate)
	default:
		return nil, fmt.Errorf("invalid rate control value: %s", rateControl)
	}

	return
}
