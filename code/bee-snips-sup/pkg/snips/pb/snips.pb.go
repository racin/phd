// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: snips.proto

package pb

import (
	fmt "fmt"
	proto "github.com/gogo/protobuf/proto"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

type Proof struct {
	Nonce  uint64 `protobuf:"varint,1,opt,name=Nonce,proto3" json:"Nonce,omitempty"`
	MPHF   []byte `protobuf:"bytes,2,opt,name=MPHF,proto3" json:"MPHF,omitempty"`
	Length uint32 `protobuf:"varint,3,opt,name=Length,proto3" json:"Length,omitempty"`
	Begin  []byte `protobuf:"bytes,4,opt,name=Begin,proto3" json:"Begin,omitempty"`
	End    []byte `protobuf:"bytes,5,opt,name=End,proto3" json:"End,omitempty"`
}

func (m *Proof) Reset()         { *m = Proof{} }
func (m *Proof) String() string { return proto.CompactTextString(m) }
func (*Proof) ProtoMessage()    {}
func (*Proof) Descriptor() ([]byte, []int) {
	return fileDescriptor_8b84dbef3c8de712, []int{0}
}
func (m *Proof) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Proof) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Proof.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Proof) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Proof.Merge(m, src)
}
func (m *Proof) XXX_Size() int {
	return m.Size()
}
func (m *Proof) XXX_DiscardUnknown() {
	xxx_messageInfo_Proof.DiscardUnknown(m)
}

var xxx_messageInfo_Proof proto.InternalMessageInfo

func (m *Proof) GetNonce() uint64 {
	if m != nil {
		return m.Nonce
	}
	return 0
}

func (m *Proof) GetMPHF() []byte {
	if m != nil {
		return m.MPHF
	}
	return nil
}

func (m *Proof) GetLength() uint32 {
	if m != nil {
		return m.Length
	}
	return 0
}

func (m *Proof) GetBegin() []byte {
	if m != nil {
		return m.Begin
	}
	return nil
}

func (m *Proof) GetEnd() []byte {
	if m != nil {
		return m.End
	}
	return nil
}

type Maintain struct {
	Nonce     uint64 `protobuf:"varint,1,opt,name=Nonce,proto3" json:"Nonce,omitempty"`
	BitVector []byte `protobuf:"bytes,2,opt,name=BitVector,proto3" json:"BitVector,omitempty"`
}

func (m *Maintain) Reset()         { *m = Maintain{} }
func (m *Maintain) String() string { return proto.CompactTextString(m) }
func (*Maintain) ProtoMessage()    {}
func (*Maintain) Descriptor() ([]byte, []int) {
	return fileDescriptor_8b84dbef3c8de712, []int{1}
}
func (m *Maintain) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Maintain) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Maintain.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Maintain) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Maintain.Merge(m, src)
}
func (m *Maintain) XXX_Size() int {
	return m.Size()
}
func (m *Maintain) XXX_DiscardUnknown() {
	xxx_messageInfo_Maintain.DiscardUnknown(m)
}

var xxx_messageInfo_Maintain proto.InternalMessageInfo

func (m *Maintain) GetNonce() uint64 {
	if m != nil {
		return m.Nonce
	}
	return 0
}

func (m *Maintain) GetBitVector() []byte {
	if m != nil {
		return m.BitVector
	}
	return nil
}

type UploadDone struct {
}

func (m *UploadDone) Reset()         { *m = UploadDone{} }
func (m *UploadDone) String() string { return proto.CompactTextString(m) }
func (*UploadDone) ProtoMessage()    {}
func (*UploadDone) Descriptor() ([]byte, []int) {
	return fileDescriptor_8b84dbef3c8de712, []int{2}
}
func (m *UploadDone) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *UploadDone) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_UploadDone.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *UploadDone) XXX_Merge(src proto.Message) {
	xxx_messageInfo_UploadDone.Merge(m, src)
}
func (m *UploadDone) XXX_Size() int {
	return m.Size()
}
func (m *UploadDone) XXX_DiscardUnknown() {
	xxx_messageInfo_UploadDone.DiscardUnknown(m)
}

var xxx_messageInfo_UploadDone proto.InternalMessageInfo

type Request struct {
	Nonce uint64 `protobuf:"varint,1,opt,name=Nonce,proto3" json:"Nonce,omitempty"`
}

func (m *Request) Reset()         { *m = Request{} }
func (m *Request) String() string { return proto.CompactTextString(m) }
func (*Request) ProtoMessage()    {}
func (*Request) Descriptor() ([]byte, []int) {
	return fileDescriptor_8b84dbef3c8de712, []int{3}
}
func (m *Request) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Request) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Request.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Request) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Request.Merge(m, src)
}
func (m *Request) XXX_Size() int {
	return m.Size()
}
func (m *Request) XXX_DiscardUnknown() {
	xxx_messageInfo_Request.DiscardUnknown(m)
}

var xxx_messageInfo_Request proto.InternalMessageInfo

func (m *Request) GetNonce() uint64 {
	if m != nil {
		return m.Nonce
	}
	return 0
}

type SignedProof struct {
	Proof     *Proof `protobuf:"bytes,1,opt,name=Proof,proto3" json:"Proof,omitempty"`
	Signature []byte `protobuf:"bytes,2,opt,name=Signature,proto3" json:"Signature,omitempty"`
	BlockHash []byte `protobuf:"bytes,3,opt,name=BlockHash,proto3" json:"BlockHash,omitempty"`
}

func (m *SignedProof) Reset()         { *m = SignedProof{} }
func (m *SignedProof) String() string { return proto.CompactTextString(m) }
func (*SignedProof) ProtoMessage()    {}
func (*SignedProof) Descriptor() ([]byte, []int) {
	return fileDescriptor_8b84dbef3c8de712, []int{4}
}
func (m *SignedProof) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *SignedProof) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_SignedProof.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *SignedProof) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SignedProof.Merge(m, src)
}
func (m *SignedProof) XXX_Size() int {
	return m.Size()
}
func (m *SignedProof) XXX_DiscardUnknown() {
	xxx_messageInfo_SignedProof.DiscardUnknown(m)
}

var xxx_messageInfo_SignedProof proto.InternalMessageInfo

func (m *SignedProof) GetProof() *Proof {
	if m != nil {
		return m.Proof
	}
	return nil
}

func (m *SignedProof) GetSignature() []byte {
	if m != nil {
		return m.Signature
	}
	return nil
}

func (m *SignedProof) GetBlockHash() []byte {
	if m != nil {
		return m.BlockHash
	}
	return nil
}

type ReqAddresses struct {
}

func (m *ReqAddresses) Reset()         { *m = ReqAddresses{} }
func (m *ReqAddresses) String() string { return proto.CompactTextString(m) }
func (*ReqAddresses) ProtoMessage()    {}
func (*ReqAddresses) Descriptor() ([]byte, []int) {
	return fileDescriptor_8b84dbef3c8de712, []int{5}
}
func (m *ReqAddresses) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ReqAddresses) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ReqAddresses.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ReqAddresses) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ReqAddresses.Merge(m, src)
}
func (m *ReqAddresses) XXX_Size() int {
	return m.Size()
}
func (m *ReqAddresses) XXX_DiscardUnknown() {
	xxx_messageInfo_ReqAddresses.DiscardUnknown(m)
}

var xxx_messageInfo_ReqAddresses proto.InternalMessageInfo

type Addresses struct {
	Marshalled []byte `protobuf:"bytes,1,opt,name=Marshalled,proto3" json:"Marshalled,omitempty"`
}

func (m *Addresses) Reset()         { *m = Addresses{} }
func (m *Addresses) String() string { return proto.CompactTextString(m) }
func (*Addresses) ProtoMessage()    {}
func (*Addresses) Descriptor() ([]byte, []int) {
	return fileDescriptor_8b84dbef3c8de712, []int{6}
}
func (m *Addresses) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Addresses) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Addresses.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Addresses) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Addresses.Merge(m, src)
}
func (m *Addresses) XXX_Size() int {
	return m.Size()
}
func (m *Addresses) XXX_DiscardUnknown() {
	xxx_messageInfo_Addresses.DiscardUnknown(m)
}

var xxx_messageInfo_Addresses proto.InternalMessageInfo

func (m *Addresses) GetMarshalled() []byte {
	if m != nil {
		return m.Marshalled
	}
	return nil
}

func init() {
	proto.RegisterType((*Proof)(nil), "snips.Proof")
	proto.RegisterType((*Maintain)(nil), "snips.Maintain")
	proto.RegisterType((*UploadDone)(nil), "snips.UploadDone")
	proto.RegisterType((*Request)(nil), "snips.Request")
	proto.RegisterType((*SignedProof)(nil), "snips.SignedProof")
	proto.RegisterType((*ReqAddresses)(nil), "snips.ReqAddresses")
	proto.RegisterType((*Addresses)(nil), "snips.Addresses")
}

func init() { proto.RegisterFile("snips.proto", fileDescriptor_8b84dbef3c8de712) }

var fileDescriptor_8b84dbef3c8de712 = []byte{
	// 314 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x74, 0x91, 0xc1, 0x4e, 0x02, 0x31,
	0x10, 0x86, 0x29, 0xb0, 0x28, 0xc3, 0x6a, 0x4c, 0x63, 0xcc, 0x1e, 0x48, 0x25, 0x3d, 0x91, 0x98,
	0x70, 0xd0, 0xbb, 0x89, 0x44, 0x0d, 0x07, 0x31, 0xa4, 0x46, 0x0f, 0xde, 0x0a, 0x3b, 0xc2, 0xc6,
	0xb5, 0x5d, 0xda, 0xf2, 0x1e, 0x3e, 0x96, 0x47, 0x8e, 0x1e, 0x0d, 0xfb, 0x22, 0x66, 0xbb, 0x0b,
	0x7a, 0xd0, 0xdb, 0xfc, 0x5f, 0xa6, 0x33, 0xff, 0xfc, 0x85, 0x8e, 0x55, 0x49, 0x66, 0x07, 0x99,
	0xd1, 0x4e, 0xd3, 0xc0, 0x0b, 0xbe, 0x84, 0x60, 0x62, 0xb4, 0x7e, 0xa1, 0xc7, 0x10, 0xdc, 0x6b,
	0x35, 0xc3, 0x88, 0xf4, 0x48, 0xbf, 0x29, 0x4a, 0x41, 0x29, 0x34, 0xc7, 0x93, 0xd1, 0x6d, 0x54,
	0xef, 0x91, 0x7e, 0x28, 0x7c, 0x4d, 0x4f, 0xa0, 0x75, 0x87, 0x6a, 0xee, 0x16, 0x51, 0xa3, 0x47,
	0xfa, 0x07, 0xa2, 0x52, 0xc5, 0x84, 0x21, 0xce, 0x13, 0x15, 0x35, 0x7d, 0x73, 0x29, 0xe8, 0x11,
	0x34, 0x6e, 0x54, 0x1c, 0x05, 0x9e, 0x15, 0x25, 0xbf, 0x84, 0xfd, 0xb1, 0x4c, 0x94, 0x93, 0x89,
	0xfa, 0x67, 0x6b, 0x17, 0xda, 0xc3, 0xc4, 0x3d, 0xe1, 0xcc, 0x69, 0x53, 0xad, 0xfe, 0x01, 0x3c,
	0x04, 0x78, 0xcc, 0x52, 0x2d, 0xe3, 0x6b, 0xad, 0x90, 0x9f, 0xc2, 0x9e, 0xc0, 0xe5, 0x0a, 0xad,
	0xfb, 0x7b, 0x18, 0x7f, 0x83, 0xce, 0x43, 0x32, 0x57, 0x18, 0x97, 0x77, 0xf2, 0xea, 0x60, 0xdf,
	0xd4, 0x39, 0x0f, 0x07, 0x65, 0x28, 0x9e, 0x89, 0x2a, 0x8b, 0x2e, 0xb4, 0x8b, 0x27, 0xd2, 0xad,
	0x0c, 0x6e, 0xf7, 0xef, 0x80, 0x77, 0x97, 0xea, 0xd9, 0xeb, 0x48, 0xda, 0x32, 0x82, 0xc2, 0xdd,
	0x16, 0xf0, 0x43, 0x08, 0x05, 0x2e, 0xaf, 0xe2, 0xd8, 0xa0, 0xb5, 0x68, 0xf9, 0x19, 0xb4, 0x77,
	0x82, 0x32, 0x80, 0xb1, 0x34, 0x76, 0x21, 0xd3, 0x14, 0x63, 0xef, 0x20, 0x14, 0xbf, 0xc8, 0xb0,
	0xfb, 0xb1, 0x61, 0x64, 0xbd, 0x61, 0xe4, 0x6b, 0xc3, 0xc8, 0x7b, 0xce, 0x6a, 0xeb, 0x9c, 0xd5,
	0x3e, 0x73, 0x56, 0x7b, 0xae, 0x67, 0xd3, 0x69, 0xcb, 0xff, 0xdc, 0xc5, 0x77, 0x00, 0x00, 0x00,
	0xff, 0xff, 0x8f, 0xf0, 0xc6, 0x69, 0xc8, 0x01, 0x00, 0x00,
}

func (m *Proof) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Proof) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Proof) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.End) > 0 {
		i -= len(m.End)
		copy(dAtA[i:], m.End)
		i = encodeVarintSnips(dAtA, i, uint64(len(m.End)))
		i--
		dAtA[i] = 0x2a
	}
	if len(m.Begin) > 0 {
		i -= len(m.Begin)
		copy(dAtA[i:], m.Begin)
		i = encodeVarintSnips(dAtA, i, uint64(len(m.Begin)))
		i--
		dAtA[i] = 0x22
	}
	if m.Length != 0 {
		i = encodeVarintSnips(dAtA, i, uint64(m.Length))
		i--
		dAtA[i] = 0x18
	}
	if len(m.MPHF) > 0 {
		i -= len(m.MPHF)
		copy(dAtA[i:], m.MPHF)
		i = encodeVarintSnips(dAtA, i, uint64(len(m.MPHF)))
		i--
		dAtA[i] = 0x12
	}
	if m.Nonce != 0 {
		i = encodeVarintSnips(dAtA, i, uint64(m.Nonce))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *Maintain) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Maintain) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Maintain) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.BitVector) > 0 {
		i -= len(m.BitVector)
		copy(dAtA[i:], m.BitVector)
		i = encodeVarintSnips(dAtA, i, uint64(len(m.BitVector)))
		i--
		dAtA[i] = 0x12
	}
	if m.Nonce != 0 {
		i = encodeVarintSnips(dAtA, i, uint64(m.Nonce))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *UploadDone) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *UploadDone) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *UploadDone) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	return len(dAtA) - i, nil
}

func (m *Request) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Request) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Request) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Nonce != 0 {
		i = encodeVarintSnips(dAtA, i, uint64(m.Nonce))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *SignedProof) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *SignedProof) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *SignedProof) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.BlockHash) > 0 {
		i -= len(m.BlockHash)
		copy(dAtA[i:], m.BlockHash)
		i = encodeVarintSnips(dAtA, i, uint64(len(m.BlockHash)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.Signature) > 0 {
		i -= len(m.Signature)
		copy(dAtA[i:], m.Signature)
		i = encodeVarintSnips(dAtA, i, uint64(len(m.Signature)))
		i--
		dAtA[i] = 0x12
	}
	if m.Proof != nil {
		{
			size, err := m.Proof.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintSnips(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *ReqAddresses) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ReqAddresses) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ReqAddresses) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	return len(dAtA) - i, nil
}

func (m *Addresses) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Addresses) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Addresses) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Marshalled) > 0 {
		i -= len(m.Marshalled)
		copy(dAtA[i:], m.Marshalled)
		i = encodeVarintSnips(dAtA, i, uint64(len(m.Marshalled)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintSnips(dAtA []byte, offset int, v uint64) int {
	offset -= sovSnips(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Proof) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Nonce != 0 {
		n += 1 + sovSnips(uint64(m.Nonce))
	}
	l = len(m.MPHF)
	if l > 0 {
		n += 1 + l + sovSnips(uint64(l))
	}
	if m.Length != 0 {
		n += 1 + sovSnips(uint64(m.Length))
	}
	l = len(m.Begin)
	if l > 0 {
		n += 1 + l + sovSnips(uint64(l))
	}
	l = len(m.End)
	if l > 0 {
		n += 1 + l + sovSnips(uint64(l))
	}
	return n
}

func (m *Maintain) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Nonce != 0 {
		n += 1 + sovSnips(uint64(m.Nonce))
	}
	l = len(m.BitVector)
	if l > 0 {
		n += 1 + l + sovSnips(uint64(l))
	}
	return n
}

func (m *UploadDone) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func (m *Request) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Nonce != 0 {
		n += 1 + sovSnips(uint64(m.Nonce))
	}
	return n
}

func (m *SignedProof) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Proof != nil {
		l = m.Proof.Size()
		n += 1 + l + sovSnips(uint64(l))
	}
	l = len(m.Signature)
	if l > 0 {
		n += 1 + l + sovSnips(uint64(l))
	}
	l = len(m.BlockHash)
	if l > 0 {
		n += 1 + l + sovSnips(uint64(l))
	}
	return n
}

func (m *ReqAddresses) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func (m *Addresses) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Marshalled)
	if l > 0 {
		n += 1 + l + sovSnips(uint64(l))
	}
	return n
}

func sovSnips(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozSnips(x uint64) (n int) {
	return sovSnips(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Proof) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowSnips
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Proof: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Proof: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Nonce", wireType)
			}
			m.Nonce = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnips
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Nonce |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MPHF", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnips
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthSnips
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthSnips
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.MPHF = append(m.MPHF[:0], dAtA[iNdEx:postIndex]...)
			if m.MPHF == nil {
				m.MPHF = []byte{}
			}
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Length", wireType)
			}
			m.Length = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnips
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Length |= uint32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Begin", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnips
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthSnips
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthSnips
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Begin = append(m.Begin[:0], dAtA[iNdEx:postIndex]...)
			if m.Begin == nil {
				m.Begin = []byte{}
			}
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field End", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnips
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthSnips
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthSnips
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.End = append(m.End[:0], dAtA[iNdEx:postIndex]...)
			if m.End == nil {
				m.End = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipSnips(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthSnips
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *Maintain) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowSnips
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Maintain: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Maintain: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Nonce", wireType)
			}
			m.Nonce = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnips
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Nonce |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field BitVector", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnips
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthSnips
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthSnips
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.BitVector = append(m.BitVector[:0], dAtA[iNdEx:postIndex]...)
			if m.BitVector == nil {
				m.BitVector = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipSnips(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthSnips
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *UploadDone) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowSnips
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: UploadDone: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: UploadDone: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		default:
			iNdEx = preIndex
			skippy, err := skipSnips(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthSnips
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *Request) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowSnips
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Request: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Request: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Nonce", wireType)
			}
			m.Nonce = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnips
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Nonce |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipSnips(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthSnips
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *SignedProof) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowSnips
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: SignedProof: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: SignedProof: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Proof", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnips
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthSnips
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthSnips
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Proof == nil {
				m.Proof = &Proof{}
			}
			if err := m.Proof.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Signature", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnips
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthSnips
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthSnips
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Signature = append(m.Signature[:0], dAtA[iNdEx:postIndex]...)
			if m.Signature == nil {
				m.Signature = []byte{}
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field BlockHash", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnips
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthSnips
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthSnips
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.BlockHash = append(m.BlockHash[:0], dAtA[iNdEx:postIndex]...)
			if m.BlockHash == nil {
				m.BlockHash = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipSnips(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthSnips
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *ReqAddresses) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowSnips
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: ReqAddresses: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ReqAddresses: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		default:
			iNdEx = preIndex
			skippy, err := skipSnips(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthSnips
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *Addresses) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowSnips
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Addresses: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Addresses: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Marshalled", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnips
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthSnips
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthSnips
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Marshalled = append(m.Marshalled[:0], dAtA[iNdEx:postIndex]...)
			if m.Marshalled == nil {
				m.Marshalled = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipSnips(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthSnips
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipSnips(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowSnips
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowSnips
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowSnips
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthSnips
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupSnips
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthSnips
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthSnips        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowSnips          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupSnips = fmt.Errorf("proto: unexpected end of group")
)
