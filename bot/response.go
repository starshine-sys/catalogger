package bot

import (
	"github.com/diamondburned/arikawa/v3/utils/httputil/httpdriver"
	"github.com/starshine-sys/catalogger/v2/bot/metrics"
	"github.com/starshine-sys/catalogger/v2/common/log"
)

// onResponse logs a request's status code and adds it to metrics
func (bot *Bot) onResponse(req httpdriver.Request, resp httpdriver.Response) error {
	method := ""

	v, ok := req.(*httpdriver.DefaultRequest)
	if ok {
		method = v.Method
		if method == "" {
			method = "GET"
		}
	}

	if resp == nil {
		return nil
	}

	if _, ok := resp.(*httpdriver.DefaultResponse); !ok {
		return nil
	}

	log.Debugf("%v %v => %v", method, metrics.LoggingName(req.GetPath()), resp.GetStatus())

	go bot.Metrics.IncRequests(method, req.GetPath(), resp.GetStatus())

	return nil
}
