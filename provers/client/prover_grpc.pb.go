// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             (unknown)
// source: prover/v1/prover.proto

package client

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	Prover_Info_FullMethodName                 = "/celestia.prover.v1.Prover/Info"
	Prover_ProveStateTransition_FullMethodName = "/celestia.prover.v1.Prover/ProveStateTransition"
	Prover_ProveStateMembership_FullMethodName = "/celestia.prover.v1.Prover/ProveStateMembership"
	Prover_VerifyProof_FullMethodName          = "/celestia.prover.v1.Prover/VerifyProof"
)

// ProverClient is the client API for Prover service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ProverClient interface {
	Info(ctx context.Context, in *InfoRequest, opts ...grpc.CallOption) (*InfoResponse, error)
	ProveStateTransition(ctx context.Context, in *ProveStateTransitionRequest, opts ...grpc.CallOption) (*ProveStateTransitionResponse, error)
	ProveStateMembership(ctx context.Context, in *ProveStateMembershipRequest, opts ...grpc.CallOption) (*ProveStateMembershipResponse, error)
	VerifyProof(ctx context.Context, in *VerifyProofRequest, opts ...grpc.CallOption) (*VerifyProofResponse, error)
}

type proverClient struct {
	cc grpc.ClientConnInterface
}

func NewProverClient(cc grpc.ClientConnInterface) ProverClient {
	return &proverClient{cc}
}

func (c *proverClient) Info(ctx context.Context, in *InfoRequest, opts ...grpc.CallOption) (*InfoResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(InfoResponse)
	err := c.cc.Invoke(ctx, Prover_Info_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *proverClient) ProveStateTransition(ctx context.Context, in *ProveStateTransitionRequest, opts ...grpc.CallOption) (*ProveStateTransitionResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ProveStateTransitionResponse)
	err := c.cc.Invoke(ctx, Prover_ProveStateTransition_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *proverClient) ProveStateMembership(ctx context.Context, in *ProveStateMembershipRequest, opts ...grpc.CallOption) (*ProveStateMembershipResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ProveStateMembershipResponse)
	err := c.cc.Invoke(ctx, Prover_ProveStateMembership_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *proverClient) VerifyProof(ctx context.Context, in *VerifyProofRequest, opts ...grpc.CallOption) (*VerifyProofResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(VerifyProofResponse)
	err := c.cc.Invoke(ctx, Prover_VerifyProof_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ProverServer is the server API for Prover service.
// All implementations must embed UnimplementedProverServer
// for forward compatibility.
type ProverServer interface {
	Info(context.Context, *InfoRequest) (*InfoResponse, error)
	ProveStateTransition(context.Context, *ProveStateTransitionRequest) (*ProveStateTransitionResponse, error)
	ProveStateMembership(context.Context, *ProveStateMembershipRequest) (*ProveStateMembershipResponse, error)
	VerifyProof(context.Context, *VerifyProofRequest) (*VerifyProofResponse, error)
	mustEmbedUnimplementedProverServer()
}

// UnimplementedProverServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedProverServer struct{}

func (UnimplementedProverServer) Info(context.Context, *InfoRequest) (*InfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Info not implemented")
}
func (UnimplementedProverServer) ProveStateTransition(context.Context, *ProveStateTransitionRequest) (*ProveStateTransitionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ProveStateTransition not implemented")
}
func (UnimplementedProverServer) ProveStateMembership(context.Context, *ProveStateMembershipRequest) (*ProveStateMembershipResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ProveStateMembership not implemented")
}
func (UnimplementedProverServer) VerifyProof(context.Context, *VerifyProofRequest) (*VerifyProofResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method VerifyProof not implemented")
}
func (UnimplementedProverServer) mustEmbedUnimplementedProverServer() {}
func (UnimplementedProverServer) testEmbeddedByValue()                {}

// UnsafeProverServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ProverServer will
// result in compilation errors.
type UnsafeProverServer interface {
	mustEmbedUnimplementedProverServer()
}

func RegisterProverServer(s grpc.ServiceRegistrar, srv ProverServer) {
	// If the following call pancis, it indicates UnimplementedProverServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&Prover_ServiceDesc, srv)
}

func _Prover_Info_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(InfoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProverServer).Info(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Prover_Info_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProverServer).Info(ctx, req.(*InfoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Prover_ProveStateTransition_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ProveStateTransitionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProverServer).ProveStateTransition(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Prover_ProveStateTransition_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProverServer).ProveStateTransition(ctx, req.(*ProveStateTransitionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Prover_ProveStateMembership_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ProveStateMembershipRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProverServer).ProveStateMembership(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Prover_ProveStateMembership_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProverServer).ProveStateMembership(ctx, req.(*ProveStateMembershipRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Prover_VerifyProof_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(VerifyProofRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProverServer).VerifyProof(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Prover_VerifyProof_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProverServer).VerifyProof(ctx, req.(*VerifyProofRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Prover_ServiceDesc is the grpc.ServiceDesc for Prover service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Prover_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "celestia.prover.v1.Prover",
	HandlerType: (*ProverServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Info",
			Handler:    _Prover_Info_Handler,
		},
		{
			MethodName: "ProveStateTransition",
			Handler:    _Prover_ProveStateTransition_Handler,
		},
		{
			MethodName: "ProveStateMembership",
			Handler:    _Prover_ProveStateMembership_Handler,
		},
		{
			MethodName: "VerifyProof",
			Handler:    _Prover_VerifyProof_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "prover/v1/prover.proto",
}
