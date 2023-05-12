// Copyright 2020 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package snips

import (
	m "github.com/ethersphere/bee/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	// Time
	CreateProofTime          prometheus.Histogram
	FindMissingChunksTime    prometheus.Histogram
	RetriveMissingChunksTime prometheus.Histogram
	GetMissingChunksTime     prometheus.Histogram
	CreateMPHFTime           prometheus.Histogram
	BuildBaseProofTime       prometheus.Histogram
	GetChunkProofTime        prometheus.Histogram
	GenChunkProofTime        prometheus.Histogram
	UploadMissingChunksTime  prometheus.Histogram
	verifySignatureTime      prometheus.Histogram

	// Data
	TotalChunksInProof    prometheus.Counter
	TotalChunksInMaintain prometheus.Counter
	TotalChunksUploaded   prometheus.Counter

	// Counter
	TotalCreateAndSendProofForMyChunks prometheus.Counter
	TotalSNIPSreqSent                  prometheus.Counter
	TotalSNIPSreqRecv                  prometheus.Counter
	TotalSNIPSproofSent                prometheus.Counter
	TotalSNIPSproofRecv                prometheus.Counter
	TotalSNIPSmaintainSent             prometheus.Counter
	TotalSNIPSmaintainRecv             prometheus.Counter
	TotalSNIPSuploaddoneSent           prometheus.Counter
	TotalSNIPSuploaddoneRecv           prometheus.Counter
	TotalErrored                       prometheus.Counter
	TotalProofsGenerated               prometheus.Counter
	SizeOfSNIPSsentMPHF                prometheus.Counter
	SizeOfSNIPSsentBlockhash           prometheus.Counter
	SizeOfSNIPSsentSignature           prometheus.Counter
	SizeOfSNIPSsentMaintainBV          prometheus.Counter

	SizeOfSNIPSsentbaseproof *prometheus.CounterVec

	SizeOfSNIPSsignedproof *prometheus.CounterVec
	SizeOfSNIPSmphf        *prometheus.CounterVec
	ShallowProofDepth      *prometheus.CounterVec
}

func newMetrics() Metrics {
	subsystem := "snips"

	return Metrics{
		// Time
		CreateProofTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "create_proof_time",
			Help:      "Create proof time.",
			Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 60},
		}),
		FindMissingChunksTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "find_missing_chunks_time",
			Help:      "Find missing chunks time.",
			Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 60},
		}),
		RetriveMissingChunksTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "retrieve_missing_chunks_time",
			Help:      "Retrieve missing chunks time.",
			Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 60},
		}),
		GetMissingChunksTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "get_missing_chunks_time",
			Help:      "Get missing chunks time.",
			Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 60},
		}),
		CreateMPHFTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "create_mphf_time",
			Help:      "Create MPHF time.",
			Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 60},
		}),
		BuildBaseProofTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "build_base_proof_time",
			Help:      "Build base proof time.",
			Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 60},
		}),
		GetChunkProofTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "get_chunk_proof_time",
			Help:      "Get chunk proof time.",
			Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 60},
		}),
		GenChunkProofTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "gen_chunk_proof_time",
			Help:      "Gen chunk proof time.",
			Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 60},
		}),
		UploadMissingChunksTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "upload_missing_chunks_time",
			Help:      "Upload missing chunks time.",
			Buckets:   []float64{1 * 1e-9, 10 * 1e-9, 100 * 1e-9, 1 * 1e-6, 10 * 1e-6, 100 * 1e-6, 1 * 1e-3, 10 * 1e-3, 100 * 1e-3, 1},
		}),
		verifySignatureTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "verify_signature_time",
			Help:      "Verify signature time.",
			Buckets:   []float64{1 * 1e-9, 10 * 1e-9, 100 * 1e-9, 1 * 1e-6, 10 * 1e-6, 100 * 1e-6, 1 * 1e-3, 10 * 1e-3, 100 * 1e-3, 1},
		}),

		// Data
		TotalChunksInProof: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_chunks_in_proof",
			Help:      "Total chunks in proof.",
		}),
		TotalChunksInMaintain: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_chunks_in_maintain",
			Help:      "Total chunks in maintain.",
		}),
		TotalChunksUploaded: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_chunks_uploaded",
			Help:      "Total chunks uploaded.",
		}),

		// Counter
		TotalSNIPSproofSent: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_snips_proof_sent",
			Help:      "Total SNIPS proof sent.",
		}),
		TotalSNIPSproofRecv: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_snips_proof_recv",
			Help:      "Total SNIPS proof recv.",
		}),
		TotalSNIPSmaintainSent: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_snips_maintain_sent",
			Help:      "Total SNIPS maintain sent.",
		}),
		TotalSNIPSmaintainRecv: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_snips_maintain_recv",
			Help:      "Total SNIPS maintain recv.",
		}),
		TotalSNIPSuploaddoneSent: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_snips_upload_done_sent",
			Help:      "Total SNIPS upload done sent.",
		}),
		TotalSNIPSuploaddoneRecv: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_snips_upload_done_recv",
			Help:      "Total SNIPS upload done recv.",
		}),
		TotalCreateAndSendProofForMyChunks: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_create_and_send_proof_for_my_chunks",
			Help:      "Total create and send proof for my chunks.",
		}),
		TotalSNIPSreqSent: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_snips_req_sent",
			Help:      "Total SNIPS req sent.",
		}),
		TotalSNIPSreqRecv: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_snips_req_recv",
			Help:      "Total SNIPS req recv.",
		}),
		TotalErrored: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_errored",
			Help:      "Total errored.",
		}),
		TotalProofsGenerated: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "total_proofs_generated",
			Help:      "Total proofs generated.",
		}),
		SizeOfSNIPSsentMPHF: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "size_of_snips_sent_mphf",
			Help:      "Size of SNIPS sent MPHF.",
		}),
		SizeOfSNIPSsentBlockhash: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "size_of_snips_sent_blockhash",
			Help:      "Size of SNIPS sent Blockhash.",
		}),
		SizeOfSNIPSsentSignature: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "size_of_snips_sent_signature",
			Help:      "Size of SNIPS sent Signature.",
		}),
		SizeOfSNIPSsentMaintainBV: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace,
			Subsystem: subsystem,
			Name:      "size_of_snips_sent_maintain_bv",
			Help:      "Size of SNIPS sent Maintain bitvector.",
		}),
		SizeOfSNIPSsentbaseproof: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "size_of_snips_sent_base_proof",
				Help:      "Size of SNIPS sent base proof.",
			},
			[]string{"nonce"},
		),
		SizeOfSNIPSsignedproof: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "size_of_snips_signed_proof",
				Help:      "Size of SNIPS signed proof.",
			},
			[]string{"nonce"},
		),
		SizeOfSNIPSmphf: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "size_of_snips_mphf",
				Help:      "Size of SNIPS mphf.",
			},
			[]string{"nonce"},
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
	}
}

func (s *Service) Metrics() []prometheus.Collector {
	return m.PrometheusCollectorsFromFields(s.metrics)
}

func (s *Service) MetricsRaw() *Metrics {
	return &s.metrics
}
