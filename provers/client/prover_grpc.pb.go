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
	Prover_ProveStateTransition_FullMethodName = "/celestia.prover.v1.Prover/ProveStateTransition"
	Prover_ProveMembership_FullMethodName      = "/celestia.prover.v1.Prover/ProveMembership"
)

// ProverClient is the client API for Prover service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ProverClient interface {
	ProveStateTransition(ctx context.Context, in *ProveStateTransitionRequest, opts ...grpc.CallOption) (*ProveStateTransitionResponse, error)
	ProveMembership(ctx context.Context, in *ProveMembershipRequest, opts ...grpc.CallOption) (*ProveMembershipResponse, error)
}

type proverClient struct {
	cc grpc.ClientConnInterface
}

func NewProverClient(cc grpc.ClientConnInterface) ProverClient {
	return &proverClient{cc}
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

func (c *proverClient) ProveMembership(ctx context.Context, in *ProveMembershipRequest, opts ...grpc.CallOption) (*ProveMembershipResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ProveMembershipResponse)
	err := c.cc.Invoke(ctx, Prover_ProveMembership_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ProverServer is the server API for Prover service.
// All implementations must embed UnimplementedProverServer
// for forward compatibility.
type ProverServer interface {
	ProveStateTransition(context.Context, *ProveStateTransitionRequest) (*ProveStateTransitionResponse, error)
	ProveMembership(context.Context, *ProveMembershipRequest) (*ProveMembershipResponse, error)
	mustEmbedUnimplementedProverServer()
}

// UnimplementedProverServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedProverServer struct{}

func (UnimplementedProverServer) ProveStateTransition(context.Context, *ProveStateTransitionRequest) (*ProveStateTransitionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ProveStateTransition not implemented")
}
func (UnimplementedProverServer) ProveMembership(context.Context, *ProveMembershipRequest) (*ProveMembershipResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ProveMembership not implemented")
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

func _Prover_ProveMembership_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ProveMembershipRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProverServer).ProveMembership(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Prover_ProveMembership_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProverServer).ProveMembership(ctx, req.(*ProveMembershipRequest))
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
			MethodName: "ProveStateTransition",
			Handler:    _Prover_ProveStateTransition_Handler,
		},
		{
			MethodName: "ProveMembership",
			Handler:    _Prover_ProveMembership_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "prover/v1/prover.proto",
}