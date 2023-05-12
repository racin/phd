// Copyright 2020 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pos

import (
	m "github.com/ethersphere/bee/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	// challenger-side

	TotalChunks                prometheus.Counter
	TotalChallenges            prometheus.Counter
	TotalProved                prometheus.Counter
	TotalReuploaded            prometheus.Counter
	TotalErrored               prometheus.Counter
	ProvedTime                 prometheus.Histogram
	ReuploadTime               prometheus.Histogram
	ErrorTime                  prometheus.Histogram
	GetChunkAddrsTime          prometheus.Histogram
	CreateChallengeTime        prometheus.Histogram
	ProveChunkTimeNoSignature  prometheus.Histogram
	ProveChunkTime             prometheus.Histogram
	VerifyProofTimeNoSignature prometheus.Histogram
	VerifyProofTime            prometheus.Histogram
	ProofDepth                 *prometheus.CounterVec
	ShallowProofDepth          *prometheus.CounterVec

	// prover-side

	TotalChallengesRecv  prometheus.Counter
	TotalForwarded       prometheus.Counter
	TotalProofsGenerated prometheus.Counter
}

func newMetrics() metrics {
	subsystem := "pos"

	return metrics{
		TotalChunks: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_chunks",
			Help:      "Total chunks to reupload.",
		}),
		TotalChallenges: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_challenges",
			Help:      "Total challenges sent.",
		}),
		TotalReuploaded: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_reuploaded",
			Help:      "Total chunks reuploaded.",
		}),
		TotalProved: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_proved",
			Help:      "Total chunks proved.",
		}),
		TotalErrored: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_errored",
			Help:      "Total chunks errored.",
		}),
		ProvedTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "proof_time",
			Help:      "Histogram of time spent to obtain a proof for a chunk.",
			Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 60},
		}),
		ReuploadTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "reupload_time",
			Help:      "Histogram of time spent to reupload a chunk.",
			Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 60},
		}),
		ErrorTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "error_time",
			Help:      "Histogram of time spent before giving up on syncing a chunk.",
			Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 60},
		}),
		GetChunkAddrsTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "getchunkaddr_time",
			Help:      "Histogram of time spent to get the addresses of all stored chunks.",
			Buckets:   []float64{1 * 1e-9, 10 * 1e-9, 100 * 1e-9, 1 * 1e-6, 10 * 1e-6, 100 * 1e-6, 1 * 1e-3, 10 * 1e-3, 100 * 1e-3, 1},
		}),
		CreateChallengeTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "createchallenge_time",
			Help:      "Histogram of time spent to create a challenge for a chunk.",
			Buckets:   []float64{1 * 1e-9, 10 * 1e-9, 100 * 1e-9, 1 * 1e-6, 10 * 1e-6, 100 * 1e-6, 1 * 1e-3, 10 * 1e-3, 100 * 1e-3, 1},
		}),
		ProveChunkTimeNoSignature: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "provechunk_nosignature_time",
			Help:      "Histogram of time spent to prove possession of a chunk without signing the message.",
			Buckets:   []float64{1 * 1e-9, 10 * 1e-9, 100 * 1e-9, 1 * 1e-6, 10 * 1e-6, 100 * 1e-6, 1 * 1e-3, 10 * 1e-3, 100 * 1e-3, 1},
		}),
		ProveChunkTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "provechunk_time",
			Help:      "Histogram of time spent to prove possession of a chunk.",
			Buckets:   []float64{1 * 1e-9, 10 * 1e-9, 100 * 1e-9, 1 * 1e-6, 10 * 1e-6, 100 * 1e-6, 1 * 1e-3, 10 * 1e-3, 100 * 1e-3, 1},
		}),
		VerifyProofTimeNoSignature: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "verifyproof_nosignature_time",
			Help:      "Histogram of time spent to verify a proof without checking the signature.",
			Buckets:   []float64{1 * 1e-9, 10 * 1e-9, 100 * 1e-9, 1 * 1e-6, 10 * 1e-6, 100 * 1e-6, 1 * 1e-3, 10 * 1e-3, 100 * 1e-3, 1},
		}),
		VerifyProofTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "verifyproof_time",
			Help:      "Histogram of time spent to verify a proof.",
			Buckets:   []float64{1 * 1e-9, 10 * 1e-9, 100 * 1e-9, 1 * 1e-6, 10 * 1e-6, 100 * 1e-6, 1 * 1e-3, 10 * 1e-3, 100 * 1e-3, 1},
		}),
		ProofDepth: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "proof_depth",
				Help:      "Counter of proofs received at different depths.",
			},
			[]string{"depth"},
		),
		ShallowProofDepth: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "shallow_proof_depth",
				Help:      "Counter of shallow proofs received at different depths.",
			},
			[]string{"depth"},
		),

		// prover-side
		TotalChallengesRecv: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_challenges_received",
			Help:      "Total challenges received.",
		}),
		TotalForwarded: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_challenges_forwarded",
			Help:      "Total challenges forwarded.",
		}),
		TotalProofsGenerated: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_proofs_generated",
			Help:      "Total proofs generated.",
		}),
	}
}

func (s *Service) Metrics() []prometheus.Collector {
	return m.PrometheusCollectorsFromFields(s.metrics)
}
