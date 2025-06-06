// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.2
// 	protoc        (unknown)
// source: prover/v1/prover.proto

package client

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type InfoRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *InfoRequest) Reset() {
	*x = InfoRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_prover_v1_prover_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *InfoRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*InfoRequest) ProtoMessage() {}

func (x *InfoRequest) ProtoReflect() protoreflect.Message {
	mi := &file_prover_v1_prover_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use InfoRequest.ProtoReflect.Descriptor instead.
func (*InfoRequest) Descriptor() ([]byte, []int) {
	return file_prover_v1_prover_proto_rawDescGZIP(), []int{0}
}

type InfoResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// TODO: add more info here as is relevant such as the circuit and state machine types
	// hex-encoded state transition verifier key
	StateTransitionVerifierKey string `protobuf:"bytes,1,opt,name=state_transition_verifier_key,json=stateTransitionVerifierKey,proto3" json:"state_transition_verifier_key,omitempty"`
	// hex-encoded state membership verifier key
	StateMembershipVerifierKey string `protobuf:"bytes,2,opt,name=state_membership_verifier_key,json=stateMembershipVerifierKey,proto3" json:"state_membership_verifier_key,omitempty"`
}

func (x *InfoResponse) Reset() {
	*x = InfoResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_prover_v1_prover_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *InfoResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*InfoResponse) ProtoMessage() {}

func (x *InfoResponse) ProtoReflect() protoreflect.Message {
	mi := &file_prover_v1_prover_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use InfoResponse.ProtoReflect.Descriptor instead.
func (*InfoResponse) Descriptor() ([]byte, []int) {
	return file_prover_v1_prover_proto_rawDescGZIP(), []int{1}
}

func (x *InfoResponse) GetStateTransitionVerifierKey() string {
	if x != nil {
		return x.StateTransitionVerifierKey
	}
	return ""
}

func (x *InfoResponse) GetStateMembershipVerifierKey() string {
	if x != nil {
		return x.StateMembershipVerifierKey
	}
	return ""
}

type ProveStateTransitionRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// For EVM chains this is the Tendermint light client contract address.
	// For Tendermint chains this is the client ID.
	ClientId string `protobuf:"bytes,1,opt,name=client_id,json=clientId,proto3" json:"client_id,omitempty"`
}

func (x *ProveStateTransitionRequest) Reset() {
	*x = ProveStateTransitionRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_prover_v1_prover_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ProveStateTransitionRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProveStateTransitionRequest) ProtoMessage() {}

func (x *ProveStateTransitionRequest) ProtoReflect() protoreflect.Message {
	mi := &file_prover_v1_prover_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ProveStateTransitionRequest.ProtoReflect.Descriptor instead.
func (*ProveStateTransitionRequest) Descriptor() ([]byte, []int) {
	return file_prover_v1_prover_proto_rawDescGZIP(), []int{2}
}

func (x *ProveStateTransitionRequest) GetClientId() string {
	if x != nil {
		return x.ClientId
	}
	return ""
}

type ProveStateTransitionResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Proof        []byte `protobuf:"bytes,1,opt,name=proof,proto3" json:"proof,omitempty"`
	PublicValues []byte `protobuf:"bytes,2,opt,name=public_values,json=publicValues,proto3" json:"public_values,omitempty"`
}

func (x *ProveStateTransitionResponse) Reset() {
	*x = ProveStateTransitionResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_prover_v1_prover_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ProveStateTransitionResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProveStateTransitionResponse) ProtoMessage() {}

func (x *ProveStateTransitionResponse) ProtoReflect() protoreflect.Message {
	mi := &file_prover_v1_prover_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ProveStateTransitionResponse.ProtoReflect.Descriptor instead.
func (*ProveStateTransitionResponse) Descriptor() ([]byte, []int) {
	return file_prover_v1_prover_proto_rawDescGZIP(), []int{3}
}

func (x *ProveStateTransitionResponse) GetProof() []byte {
	if x != nil {
		return x.Proof
	}
	return nil
}

func (x *ProveStateTransitionResponse) GetPublicValues() []byte {
	if x != nil {
		return x.PublicValues
	}
	return nil
}

type ProveStateMembershipRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// For EVM chains this is the Tendermint light client contract address.
	// For Tendermint chains this is the client ID.
	ClientId string   `protobuf:"bytes,1,opt,name=client_id,json=clientId,proto3" json:"client_id,omitempty"`
	KeyPaths []string `protobuf:"bytes,2,rep,name=key_paths,json=keyPaths,proto3" json:"key_paths,omitempty"`
}

func (x *ProveStateMembershipRequest) Reset() {
	*x = ProveStateMembershipRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_prover_v1_prover_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ProveStateMembershipRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProveStateMembershipRequest) ProtoMessage() {}

func (x *ProveStateMembershipRequest) ProtoReflect() protoreflect.Message {
	mi := &file_prover_v1_prover_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ProveStateMembershipRequest.ProtoReflect.Descriptor instead.
func (*ProveStateMembershipRequest) Descriptor() ([]byte, []int) {
	return file_prover_v1_prover_proto_rawDescGZIP(), []int{4}
}

func (x *ProveStateMembershipRequest) GetClientId() string {
	if x != nil {
		return x.ClientId
	}
	return ""
}

func (x *ProveStateMembershipRequest) GetKeyPaths() []string {
	if x != nil {
		return x.KeyPaths
	}
	return nil
}

type ProveStateMembershipResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Proof  []byte `protobuf:"bytes,1,opt,name=proof,proto3" json:"proof,omitempty"`
	Height int64  `protobuf:"varint,2,opt,name=height,proto3" json:"height,omitempty"`
}

func (x *ProveStateMembershipResponse) Reset() {
	*x = ProveStateMembershipResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_prover_v1_prover_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ProveStateMembershipResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProveStateMembershipResponse) ProtoMessage() {}

func (x *ProveStateMembershipResponse) ProtoReflect() protoreflect.Message {
	mi := &file_prover_v1_prover_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ProveStateMembershipResponse.ProtoReflect.Descriptor instead.
func (*ProveStateMembershipResponse) Descriptor() ([]byte, []int) {
	return file_prover_v1_prover_proto_rawDescGZIP(), []int{5}
}

func (x *ProveStateMembershipResponse) GetProof() []byte {
	if x != nil {
		return x.Proof
	}
	return nil
}

func (x *ProveStateMembershipResponse) GetHeight() int64 {
	if x != nil {
		return x.Height
	}
	return 0
}

type VerifyProofRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Proof           []byte `protobuf:"bytes,1,opt,name=proof,proto3" json:"proof,omitempty"`
	Sp1PublicInputs []byte `protobuf:"bytes,2,opt,name=sp1_public_inputs,json=sp1PublicInputs,proto3" json:"sp1_public_inputs,omitempty"`
	Sp1VkeyHash     string `protobuf:"bytes,3,opt,name=sp1_vkey_hash,json=sp1VkeyHash,proto3" json:"sp1_vkey_hash,omitempty"`
	Groth16Vk       []byte `protobuf:"bytes,4,opt,name=groth16_vk,json=groth16Vk,proto3" json:"groth16_vk,omitempty"`
}

func (x *VerifyProofRequest) Reset() {
	*x = VerifyProofRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_prover_v1_prover_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *VerifyProofRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*VerifyProofRequest) ProtoMessage() {}

func (x *VerifyProofRequest) ProtoReflect() protoreflect.Message {
	mi := &file_prover_v1_prover_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use VerifyProofRequest.ProtoReflect.Descriptor instead.
func (*VerifyProofRequest) Descriptor() ([]byte, []int) {
	return file_prover_v1_prover_proto_rawDescGZIP(), []int{6}
}

func (x *VerifyProofRequest) GetProof() []byte {
	if x != nil {
		return x.Proof
	}
	return nil
}

func (x *VerifyProofRequest) GetSp1PublicInputs() []byte {
	if x != nil {
		return x.Sp1PublicInputs
	}
	return nil
}

func (x *VerifyProofRequest) GetSp1VkeyHash() string {
	if x != nil {
		return x.Sp1VkeyHash
	}
	return ""
}

func (x *VerifyProofRequest) GetGroth16Vk() []byte {
	if x != nil {
		return x.Groth16Vk
	}
	return nil
}

type VerifyProofResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Success      bool   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	ErrorMessage string `protobuf:"bytes,2,opt,name=error_message,json=errorMessage,proto3" json:"error_message,omitempty"` // Only set if success is false
}

func (x *VerifyProofResponse) Reset() {
	*x = VerifyProofResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_prover_v1_prover_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *VerifyProofResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*VerifyProofResponse) ProtoMessage() {}

func (x *VerifyProofResponse) ProtoReflect() protoreflect.Message {
	mi := &file_prover_v1_prover_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use VerifyProofResponse.ProtoReflect.Descriptor instead.
func (*VerifyProofResponse) Descriptor() ([]byte, []int) {
	return file_prover_v1_prover_proto_rawDescGZIP(), []int{7}
}

func (x *VerifyProofResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

func (x *VerifyProofResponse) GetErrorMessage() string {
	if x != nil {
		return x.ErrorMessage
	}
	return ""
}

var File_prover_v1_prover_proto protoreflect.FileDescriptor

var file_prover_v1_prover_proto_rawDesc = []byte{
	0x0a, 0x16, 0x70, 0x72, 0x6f, 0x76, 0x65, 0x72, 0x2f, 0x76, 0x31, 0x2f, 0x70, 0x72, 0x6f, 0x76,
	0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x12, 0x63, 0x65, 0x6c, 0x65, 0x73, 0x74,
	0x69, 0x61, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x22, 0x0d, 0x0a, 0x0b,
	0x49, 0x6e, 0x66, 0x6f, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x22, 0x94, 0x01, 0x0a, 0x0c,
	0x49, 0x6e, 0x66, 0x6f, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x41, 0x0a, 0x1d,
	0x73, 0x74, 0x61, 0x74, 0x65, 0x5f, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x69, 0x74, 0x69, 0x6f, 0x6e,
	0x5f, 0x76, 0x65, 0x72, 0x69, 0x66, 0x69, 0x65, 0x72, 0x5f, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x1a, 0x73, 0x74, 0x61, 0x74, 0x65, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x69,
	0x74, 0x69, 0x6f, 0x6e, 0x56, 0x65, 0x72, 0x69, 0x66, 0x69, 0x65, 0x72, 0x4b, 0x65, 0x79, 0x12,
	0x41, 0x0a, 0x1d, 0x73, 0x74, 0x61, 0x74, 0x65, 0x5f, 0x6d, 0x65, 0x6d, 0x62, 0x65, 0x72, 0x73,
	0x68, 0x69, 0x70, 0x5f, 0x76, 0x65, 0x72, 0x69, 0x66, 0x69, 0x65, 0x72, 0x5f, 0x6b, 0x65, 0x79,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x1a, 0x73, 0x74, 0x61, 0x74, 0x65, 0x4d, 0x65, 0x6d,
	0x62, 0x65, 0x72, 0x73, 0x68, 0x69, 0x70, 0x56, 0x65, 0x72, 0x69, 0x66, 0x69, 0x65, 0x72, 0x4b,
	0x65, 0x79, 0x22, 0x3a, 0x0a, 0x1b, 0x50, 0x72, 0x6f, 0x76, 0x65, 0x53, 0x74, 0x61, 0x74, 0x65,
	0x54, 0x72, 0x61, 0x6e, 0x73, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x12, 0x1b, 0x0a, 0x09, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x49, 0x64, 0x22, 0x59,
	0x0a, 0x1c, 0x50, 0x72, 0x6f, 0x76, 0x65, 0x53, 0x74, 0x61, 0x74, 0x65, 0x54, 0x72, 0x61, 0x6e,
	0x73, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x14,
	0x0a, 0x05, 0x70, 0x72, 0x6f, 0x6f, 0x66, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x05, 0x70,
	0x72, 0x6f, 0x6f, 0x66, 0x12, 0x23, 0x0a, 0x0d, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x5f, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x0c, 0x70, 0x75, 0x62,
	0x6c, 0x69, 0x63, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x22, 0x57, 0x0a, 0x1b, 0x50, 0x72, 0x6f,
	0x76, 0x65, 0x53, 0x74, 0x61, 0x74, 0x65, 0x4d, 0x65, 0x6d, 0x62, 0x65, 0x72, 0x73, 0x68, 0x69,
	0x70, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1b, 0x0a, 0x09, 0x63, 0x6c, 0x69, 0x65,
	0x6e, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x63, 0x6c, 0x69,
	0x65, 0x6e, 0x74, 0x49, 0x64, 0x12, 0x1b, 0x0a, 0x09, 0x6b, 0x65, 0x79, 0x5f, 0x70, 0x61, 0x74,
	0x68, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x09, 0x52, 0x08, 0x6b, 0x65, 0x79, 0x50, 0x61, 0x74,
	0x68, 0x73, 0x22, 0x4c, 0x0a, 0x1c, 0x50, 0x72, 0x6f, 0x76, 0x65, 0x53, 0x74, 0x61, 0x74, 0x65,
	0x4d, 0x65, 0x6d, 0x62, 0x65, 0x72, 0x73, 0x68, 0x69, 0x70, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x70, 0x72, 0x6f, 0x6f, 0x66, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0c, 0x52, 0x05, 0x70, 0x72, 0x6f, 0x6f, 0x66, 0x12, 0x16, 0x0a, 0x06, 0x68, 0x65, 0x69, 0x67,
	0x68, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x06, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74,
	0x22, 0x99, 0x01, 0x0a, 0x12, 0x56, 0x65, 0x72, 0x69, 0x66, 0x79, 0x50, 0x72, 0x6f, 0x6f, 0x66,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x14, 0x0a, 0x05, 0x70, 0x72, 0x6f, 0x6f, 0x66,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x05, 0x70, 0x72, 0x6f, 0x6f, 0x66, 0x12, 0x2a, 0x0a,
	0x11, 0x73, 0x70, 0x31, 0x5f, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x5f, 0x69, 0x6e, 0x70, 0x75,
	0x74, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x0f, 0x73, 0x70, 0x31, 0x50, 0x75, 0x62,
	0x6c, 0x69, 0x63, 0x49, 0x6e, 0x70, 0x75, 0x74, 0x73, 0x12, 0x22, 0x0a, 0x0d, 0x73, 0x70, 0x31,
	0x5f, 0x76, 0x6b, 0x65, 0x79, 0x5f, 0x68, 0x61, 0x73, 0x68, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x0b, 0x73, 0x70, 0x31, 0x56, 0x6b, 0x65, 0x79, 0x48, 0x61, 0x73, 0x68, 0x12, 0x1d, 0x0a,
	0x0a, 0x67, 0x72, 0x6f, 0x74, 0x68, 0x31, 0x36, 0x5f, 0x76, 0x6b, 0x18, 0x04, 0x20, 0x01, 0x28,
	0x0c, 0x52, 0x09, 0x67, 0x72, 0x6f, 0x74, 0x68, 0x31, 0x36, 0x56, 0x6b, 0x22, 0x54, 0x0a, 0x13,
	0x56, 0x65, 0x72, 0x69, 0x66, 0x79, 0x50, 0x72, 0x6f, 0x6f, 0x66, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x08, 0x52, 0x07, 0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x12, 0x23, 0x0a,
	0x0d, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x5f, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x4d, 0x65, 0x73, 0x73, 0x61,
	0x67, 0x65, 0x32, 0xa9, 0x03, 0x0a, 0x06, 0x50, 0x72, 0x6f, 0x76, 0x65, 0x72, 0x12, 0x49, 0x0a,
	0x04, 0x49, 0x6e, 0x66, 0x6f, 0x12, 0x1f, 0x2e, 0x63, 0x65, 0x6c, 0x65, 0x73, 0x74, 0x69, 0x61,
	0x2e, 0x70, 0x72, 0x6f, 0x76, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x49, 0x6e, 0x66, 0x6f, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x20, 0x2e, 0x63, 0x65, 0x6c, 0x65, 0x73, 0x74, 0x69,
	0x61, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x49, 0x6e, 0x66, 0x6f,
	0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x79, 0x0a, 0x14, 0x50, 0x72, 0x6f, 0x76,
	0x65, 0x53, 0x74, 0x61, 0x74, 0x65, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x69, 0x74, 0x69, 0x6f, 0x6e,
	0x12, 0x2f, 0x2e, 0x63, 0x65, 0x6c, 0x65, 0x73, 0x74, 0x69, 0x61, 0x2e, 0x70, 0x72, 0x6f, 0x76,
	0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x50, 0x72, 0x6f, 0x76, 0x65, 0x53, 0x74, 0x61, 0x74, 0x65,
	0x54, 0x72, 0x61, 0x6e, 0x73, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x1a, 0x30, 0x2e, 0x63, 0x65, 0x6c, 0x65, 0x73, 0x74, 0x69, 0x61, 0x2e, 0x70, 0x72, 0x6f,
	0x76, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x50, 0x72, 0x6f, 0x76, 0x65, 0x53, 0x74, 0x61, 0x74,
	0x65, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x12, 0x79, 0x0a, 0x14, 0x50, 0x72, 0x6f, 0x76, 0x65, 0x53, 0x74, 0x61, 0x74,
	0x65, 0x4d, 0x65, 0x6d, 0x62, 0x65, 0x72, 0x73, 0x68, 0x69, 0x70, 0x12, 0x2f, 0x2e, 0x63, 0x65,
	0x6c, 0x65, 0x73, 0x74, 0x69, 0x61, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x65, 0x72, 0x2e, 0x76, 0x31,
	0x2e, 0x50, 0x72, 0x6f, 0x76, 0x65, 0x53, 0x74, 0x61, 0x74, 0x65, 0x4d, 0x65, 0x6d, 0x62, 0x65,
	0x72, 0x73, 0x68, 0x69, 0x70, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x30, 0x2e, 0x63,
	0x65, 0x6c, 0x65, 0x73, 0x74, 0x69, 0x61, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x65, 0x72, 0x2e, 0x76,
	0x31, 0x2e, 0x50, 0x72, 0x6f, 0x76, 0x65, 0x53, 0x74, 0x61, 0x74, 0x65, 0x4d, 0x65, 0x6d, 0x62,
	0x65, 0x72, 0x73, 0x68, 0x69, 0x70, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x5e,
	0x0a, 0x0b, 0x56, 0x65, 0x72, 0x69, 0x66, 0x79, 0x50, 0x72, 0x6f, 0x6f, 0x66, 0x12, 0x26, 0x2e,
	0x63, 0x65, 0x6c, 0x65, 0x73, 0x74, 0x69, 0x61, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x65, 0x72, 0x2e,
	0x76, 0x31, 0x2e, 0x56, 0x65, 0x72, 0x69, 0x66, 0x79, 0x50, 0x72, 0x6f, 0x6f, 0x66, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x27, 0x2e, 0x63, 0x65, 0x6c, 0x65, 0x73, 0x74, 0x69, 0x61,
	0x2e, 0x70, 0x72, 0x6f, 0x76, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x56, 0x65, 0x72, 0x69, 0x66,
	0x79, 0x50, 0x72, 0x6f, 0x6f, 0x66, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x42, 0x10,
	0x5a, 0x0e, 0x70, 0x72, 0x6f, 0x76, 0x65, 0x72, 0x73, 0x2f, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74,
	0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_prover_v1_prover_proto_rawDescOnce sync.Once
	file_prover_v1_prover_proto_rawDescData = file_prover_v1_prover_proto_rawDesc
)

func file_prover_v1_prover_proto_rawDescGZIP() []byte {
	file_prover_v1_prover_proto_rawDescOnce.Do(func() {
		file_prover_v1_prover_proto_rawDescData = protoimpl.X.CompressGZIP(file_prover_v1_prover_proto_rawDescData)
	})
	return file_prover_v1_prover_proto_rawDescData
}

var file_prover_v1_prover_proto_msgTypes = make([]protoimpl.MessageInfo, 8)
var file_prover_v1_prover_proto_goTypes = []any{
	(*InfoRequest)(nil),                  // 0: celestia.prover.v1.InfoRequest
	(*InfoResponse)(nil),                 // 1: celestia.prover.v1.InfoResponse
	(*ProveStateTransitionRequest)(nil),  // 2: celestia.prover.v1.ProveStateTransitionRequest
	(*ProveStateTransitionResponse)(nil), // 3: celestia.prover.v1.ProveStateTransitionResponse
	(*ProveStateMembershipRequest)(nil),  // 4: celestia.prover.v1.ProveStateMembershipRequest
	(*ProveStateMembershipResponse)(nil), // 5: celestia.prover.v1.ProveStateMembershipResponse
	(*VerifyProofRequest)(nil),           // 6: celestia.prover.v1.VerifyProofRequest
	(*VerifyProofResponse)(nil),          // 7: celestia.prover.v1.VerifyProofResponse
}
var file_prover_v1_prover_proto_depIdxs = []int32{
	0, // 0: celestia.prover.v1.Prover.Info:input_type -> celestia.prover.v1.InfoRequest
	2, // 1: celestia.prover.v1.Prover.ProveStateTransition:input_type -> celestia.prover.v1.ProveStateTransitionRequest
	4, // 2: celestia.prover.v1.Prover.ProveStateMembership:input_type -> celestia.prover.v1.ProveStateMembershipRequest
	6, // 3: celestia.prover.v1.Prover.VerifyProof:input_type -> celestia.prover.v1.VerifyProofRequest
	1, // 4: celestia.prover.v1.Prover.Info:output_type -> celestia.prover.v1.InfoResponse
	3, // 5: celestia.prover.v1.Prover.ProveStateTransition:output_type -> celestia.prover.v1.ProveStateTransitionResponse
	5, // 6: celestia.prover.v1.Prover.ProveStateMembership:output_type -> celestia.prover.v1.ProveStateMembershipResponse
	7, // 7: celestia.prover.v1.Prover.VerifyProof:output_type -> celestia.prover.v1.VerifyProofResponse
	4, // [4:8] is the sub-list for method output_type
	0, // [0:4] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_prover_v1_prover_proto_init() }
func file_prover_v1_prover_proto_init() {
	if File_prover_v1_prover_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_prover_v1_prover_proto_msgTypes[0].Exporter = func(v any, i int) any {
			switch v := v.(*InfoRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_prover_v1_prover_proto_msgTypes[1].Exporter = func(v any, i int) any {
			switch v := v.(*InfoResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_prover_v1_prover_proto_msgTypes[2].Exporter = func(v any, i int) any {
			switch v := v.(*ProveStateTransitionRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_prover_v1_prover_proto_msgTypes[3].Exporter = func(v any, i int) any {
			switch v := v.(*ProveStateTransitionResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_prover_v1_prover_proto_msgTypes[4].Exporter = func(v any, i int) any {
			switch v := v.(*ProveStateMembershipRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_prover_v1_prover_proto_msgTypes[5].Exporter = func(v any, i int) any {
			switch v := v.(*ProveStateMembershipResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_prover_v1_prover_proto_msgTypes[6].Exporter = func(v any, i int) any {
			switch v := v.(*VerifyProofRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_prover_v1_prover_proto_msgTypes[7].Exporter = func(v any, i int) any {
			switch v := v.(*VerifyProofResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_prover_v1_prover_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   8,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_prover_v1_prover_proto_goTypes,
		DependencyIndexes: file_prover_v1_prover_proto_depIdxs,
		MessageInfos:      file_prover_v1_prover_proto_msgTypes,
	}.Build()
	File_prover_v1_prover_proto = out.File
	file_prover_v1_prover_proto_rawDesc = nil
	file_prover_v1_prover_proto_goTypes = nil
	file_prover_v1_prover_proto_depIdxs = nil
}
