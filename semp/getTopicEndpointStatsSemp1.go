package semp

import (
	"encoding/xml"
	"errors"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// Get rates for each individual topic-endpoint of all vpn's
// This can result in heavy system load for lots of topc-endpoints
func (e *Semp) GetTopicEndpointStatsSemp1(ch chan<- prometheus.Metric, vpnFilter string, itemFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				TopicEndpoint struct {
					TopicEndpoints struct {
						TopicEndpoint []struct {
							TopicEndointName string `xml:"name"`
							Info             struct {
								MsgVpnName string `xml:"message-vpn"`
							} `xml:"info"`
							Stats struct {
								MessageSpoolStats struct {
									TotalByteSpooled       float64 `xml:"total-bytes-spooled"`
									TotalMsgSpooled        float64 `xml:"total-messages-spooled"`
									MsgRedelivered         float64 `xml:"messages-redelivered"`
									MsgRetransmit          float64 `xml:"messages-transport-retransmit"`
									SpoolUsageExceeded     float64 `xml:"spool-usage-exceeded"`
									MsgSizeExceeded        float64 `xml:"max-message-size-exceeded"`
									SpoolShutdownDiscard   float64 `xml:"spool-shutdown-discard"`
									DestinationGroupError  float64 `xml:"destination-group-error"`
									LowPrioMsgDiscard      float64 `xml:"low-priority-msg-congestion-discard"`
									Deleted                float64 `xml:"total-deleted-messages"`
									TtlDisacarded          float64 `xml:"total-ttl-expired-discard-messages"`
									TtlDmq                 float64 `xml:"total-ttl-expired-to-dmq-messages"`
									TtlDmqFailed           float64 `xml:"total-ttl-expired-to-dmq-failures"`
									MaxRedeliveryDiscarded float64 `xml:"max-redelivery-exceeded-discard-messages"`
									MaxRedeliveryDmq       float64 `xml:"max-redelivery-exceeded-to-dmq-messages"`
									MaxRedeliveryDmqFailed float64 `xml:"max-redelivery-exceeded-to-dmq-failures"`
								} `xml:"message-spool-stats"`
							} `xml:"stats"`
						} `xml:"topic-endpoint"`
					} `xml:"topic-endpoints"`
				} `xml:"topic-endpoint"`
			} `xml:"show"`
		} `xml:"rpc"`
		MoreCookie struct {
			RPC string `xml:",innerxml"`
		} `xml:"more-cookie"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	var page = 1
	var lastTopicEndpointName = ""
	for nextRequest := "<rpc><show><topic-endpoint><name>" + itemFilter + "</name><vpn-name>" + vpnFilter + "</vpn-name><stats/><count/><num-elements>100</num-elements></topic-endpoint></show></rpc>"; nextRequest != ""; {
		body, err := e.postHTTP(e.brokerURI+"/SEMP", "application/xml", nextRequest, "TopicEndpointStatsSemp1", page)
		page++

		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't scrape TopicEndpointStatsSemp1", "err", err, "broker", e.brokerURI)
			return 0, err
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't decode TopicEndpointStatsSemp1", "err", err, "broker", e.brokerURI)
			return 0, err
		}
		if target.ExecuteResult.Result != "ok" {
			_ = level.Error(e.logger).Log("msg", "unexpected result", "command", nextRequest, "result", target.ExecuteResult.Result, "broker", e.brokerURI)
			return 0, errors.New("unexpected result: see log")
		}

		//fmt.Printf("Next request: %v\n", target.MoreCookie.RPC)
		nextRequest = target.MoreCookie.RPC

		for _, topicEndpoint := range target.RPC.Show.TopicEndpoint.TopicEndpoints.TopicEndpoint {
			topicEndpointKey := topicEndpoint.Info.MsgVpnName + "___" + topicEndpoint.TopicEndointName
			if topicEndpointKey == lastTopicEndpointName {
				continue
			}
			lastTopicEndpointName = topicEndpointKey
			ch <- prometheus.MustNewConstMetric(MetricDesc["TopicEndpointStats"]["total_bytes_spooled"], prometheus.GaugeValue, topicEndpoint.Stats.MessageSpoolStats.TotalByteSpooled, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
			ch <- prometheus.MustNewConstMetric(MetricDesc["TopicEndpointStats"]["total_messages_spooled"], prometheus.GaugeValue, topicEndpoint.Stats.MessageSpoolStats.TotalMsgSpooled, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
			ch <- prometheus.MustNewConstMetric(MetricDesc["TopicEndpointStats"]["messages_redelivered"], prometheus.GaugeValue, topicEndpoint.Stats.MessageSpoolStats.MsgRedelivered, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
			ch <- prometheus.MustNewConstMetric(MetricDesc["TopicEndpointStats"]["messages_transport_retransmited"], prometheus.GaugeValue, topicEndpoint.Stats.MessageSpoolStats.MsgRetransmit, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
			ch <- prometheus.MustNewConstMetric(MetricDesc["TopicEndpointStats"]["spool_usage_exceeded"], prometheus.GaugeValue, topicEndpoint.Stats.MessageSpoolStats.SpoolUsageExceeded, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
			ch <- prometheus.MustNewConstMetric(MetricDesc["TopicEndpointStats"]["max_message_size_exceeded"], prometheus.GaugeValue, topicEndpoint.Stats.MessageSpoolStats.MsgSizeExceeded, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
			ch <- prometheus.MustNewConstMetric(MetricDesc["TopicEndpointStats"]["total_deleted_messages"], prometheus.GaugeValue, topicEndpoint.Stats.MessageSpoolStats.Deleted, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
		}
		body.Close()
	}

	return 1, nil
}
