package quota

import (
	"git.code.oa.com/polaris/polaris-go/pkg/log"
	"git.code.oa.com/polaris/polaris-go/pkg/model"
	rlimitV2 "git.code.oa.com/polaris/polaris-go/pkg/model/pb/metric/v2"
	namingpb "git.code.oa.com/polaris/polaris-go/pkg/model/pb/v1"
	"time"
)

// 异步处理发送init
func (r *RateLimitWindow) DoAsyncRemoteInit() error {
	if r.Rule.GetType() == namingpb.Rule_LOCAL || r.configMode == model.ConfigQuotaLocalMode {
		return nil
	}
	sender, err := r.AsyncRateLimitConnector().GetMessageSender(r.remoteCluster, r.hashValue)
	if nil != err {
		log.GetBaseLogger().Errorf("fail to call RateLimitService.GetMessageSender, service %s, error is %s",
			r.remoteCluster, err)
		return err
	}
	timeDiff := sender.AdjustTime()
	r.allocatingBucket.UpdateTimeDiff(timeDiff)

	request := r.InitializeRequest()
	sender.SendInitRequest(request, r)
	return nil
}

// 异步发送 acquire
func (r *RateLimitWindow) DoAsyncRemoteAcquire() error {
	if r.Rule.GetType() == namingpb.Rule_LOCAL || r.configMode == model.ConfigQuotaLocalMode {
		return nil
	}
	sender, err := r.AsyncRateLimitConnector().GetMessageSender(r.remoteCluster, r.hashValue)
	if nil != err {
		log.GetBaseLogger().Errorf(
			"fail to call RateLimitService.GetMessageSender, service %s, error is %s",
			r.remoteCluster, err)
		return err
	}
	if !sender.HasInitialized(r.SvcKey, r.Labels) {
		r.SetStatus(Initializing)
		return r.DoAsyncRemoteInit()
	}

	timeDiff := sender.AdjustTime()
	r.allocatingBucket.UpdateTimeDiff(timeDiff)

	request := r.acquireRequest()
	err = sender.SendReportRequest(request)
	if nil != err {
		log.GetBaseLogger().Errorf(
			"fail to call RateLimitService.Acquire, service %s, labels %s, error is %s",
			r.SvcKey, r.Labels, err)
		return err
	}
	return nil
}

//应答回调函数
func (r *RateLimitWindow) OnInitResponse(counter *rlimitV2.QuotaCounter, duration time.Duration, srvTimeMilli int64) {
	r.SetStatus(Initialized)
	log.GetBaseLogger().Infof("[RateLimit]window %s changed to initialized", r.uniqueKey)
	r.allocatingBucket.SetRemoteQuota(&RemoteQuotaResult{
		Left:            counter.GetLeft(),
		ClientCount:     counter.GetClientCount(),
		ServerTimeMilli: srvTimeMilli,
		DurationMill:    model.ToMilliSeconds(duration),
	})
}

//应答回调函数
func (r *RateLimitWindow) OnReportResponse(counter *rlimitV2.QuotaLeft, duration time.Duration, curTimeMilli int64) {
	r.allocatingBucket.SetRemoteQuota(&RemoteQuotaResult{
		Left:            counter.GetLeft(),
		ClientCount:     counter.GetClientCount(),
		ServerTimeMilli: curTimeMilli,
		DurationMill:    model.ToMilliSeconds(duration),
	})
}
