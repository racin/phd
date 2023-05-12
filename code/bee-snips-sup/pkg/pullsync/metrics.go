// Copyright 2020 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pullsync

import (
	m "github.com/ethersphere/bee/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	Offered       prometheus.Counter // number of chunks offered
	Wanted        prometheus.Counter // number of chunks wanted
	Delivered     prometheus.Counter // number of chunk deliveries
	DbOps         prometheus.Counter // number of db ops
	DuplicateRuid prometheus.Counter //number of duplicate RUID requests we got

	// Racin metrics

	// Time
	ProcessWantTime  prometheus.Histogram // time to process a want request
	UploadChunksTime prometheus.Histogram // time to upload chunks
	MakeOfferTime    prometheus.Histogram // time to make an offer

	// Counters
	TotalCursorReqSent          prometheus.Counter // total number of cursor requests sent
	TotalCursorReqRecv          prometheus.Counter // total number of cursor requests received
	TotalCursorSent             prometheus.Counter // total number of cursors sent
	TotalCursorRecv             prometheus.Counter // total number of cursors received
	TotalRuidSent               prometheus.Counter // number of RUIDs sent
	TotalRuidRecv               prometheus.Counter // number of RUIDs received
	TotalRangeSent              prometheus.Counter // number of ranges sent
	TotalRangeRecv              prometheus.Counter // number of ranges recv
	TotalOfferSent              prometheus.Counter // number of offers sent
	TotalOfferRecv              prometheus.Counter // number of offers received
	TotalWantSent               prometheus.Counter // number of Wants sent
	TotalWantRecv               prometheus.Counter // number of Wants received
	TotalInconsistentHashLength prometheus.Counter // number of errors with inconsistent hash length
	TotalZeroHashes             prometheus.Counter // numer of errors with zero hashes

	SizeOfWantSent   prometheus.Counter // size of want sent
	SizeOfCursorSent prometheus.Counter // size of cursor sent

	// Data
	TotalChunksInOffer prometheus.Counter // number of chunks in offer
	TotalChunksInWant  prometheus.Counter // number of chunks in want

}

func newMetrics() metrics {
	subsystem := "pullsync"

	return metrics{
		Offered: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "chunks_offered",
			Help:      "Total chunks offered.",
		}),
		Wanted: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "chunks_wanted",
			Help:      "Total chunks wanted.",
		}),
		Delivered: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "chunks_delivered",
			Help:      "Total chunks delivered.",
		}),
		DbOps: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "db_ops",
			Help:      "Total Db Ops.",
		}),
		DuplicateRuid: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "duplicate_ruids",
			Help:      "Total duplicate RUIDs.",
		}),
		// Racin metrics
		ProcessWantTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "process_want_time",
			Help:      "Process want time.",
			Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 60},
		}),
		UploadChunksTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "upload_chunks_time",
			Help:      "Upload chunks time.",
			Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 60},
		}),
		MakeOfferTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "make_offer_time",
			Help:      "Make offer time.",
			Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 60},
		}),
		TotalCursorReqSent: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_cursor_req_sent",
			Help:      "Total Cursors requests sent.",
		}),
		TotalCursorReqRecv: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_cursor_req_recv",
			Help:      "Total Cursors requests received.",
		}),
		TotalCursorSent: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_cursor_sent",
			Help:      "Total Cursors sent.",
		}),
		TotalCursorRecv: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_cursor_recv",
			Help:      "Total Cursors received.",
		}),
		TotalRuidSent: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_ruid_sent",
			Help:      "Total RUIDs sent.",
		}),
		TotalRuidRecv: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_ruid_recv",
			Help:      "Total RUIDs received.",
		}),
		TotalRangeSent: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_range_sent",
			Help:      "Total Range Sent.",
		}),
		TotalRangeRecv: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_range_recv",
			Help:      "Total Range recv.",
		}),
		TotalOfferSent: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_offer_sent",
			Help:      "Total Offers sent.",
		}),
		TotalOfferRecv: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_offer_recv",
			Help:      "Total Offer Recv.",
		}),
		TotalWantSent: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_want_sent",
			Help:      "Total Want sent.",
		}),
		TotalWantRecv: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_want_recv",
			Help:      "Total Want Recv.",
		}),
		TotalInconsistentHashLength: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_inconsistent_hash_length",
			Help:      "Total Inconsistent Hash Length.",
		}),
		TotalZeroHashes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_zero_hashes",
			Help:      "Total Zero hashes.",
		}),
		SizeOfWantSent: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "size_of_want_sent",
			Help:      "Size of Want sent.",
		}),
		SizeOfCursorSent: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "size_of_cursor_sent",
			Help:      "Size of Cursor sent.",
		}),
		TotalChunksInOffer: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_chunks_in_offer",
			Help:      "Total Chunks in Offer.",
		}),
		TotalChunksInWant: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_chunks_in_want",
			Help:      "Total Chunks In Want.",
		}),
	}
}

func (s *Syncer) Metrics() []prometheus.Collector {
	return m.PrometheusCollectorsFromFields(s.metrics)
}
