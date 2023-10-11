// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.20.3
// source: protos/cluster.proto

package protos

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// ImmutableStorageClusterClient is the client API for ImmutableStorageCluster service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ImmutableStorageClusterClient interface {
	PropagateWrite(ctx context.Context, in *PropagateWriteRequest, opts ...grpc.CallOption) (*PropagateWriteResponse, error)
	Sync(ctx context.Context, in *SyncRequest, opts ...grpc.CallOption) (*SyncResponse, error)
	GetNodetxtIPNS(ctx context.Context, in *GetNodetxtIPNSRequest, opts ...grpc.CallOption) (*GetNodetxtIPNSResponse, error)
}

type immutableStorageClusterClient struct {
	cc grpc.ClientConnInterface
}

func NewImmutableStorageClusterClient(cc grpc.ClientConnInterface) ImmutableStorageClusterClient {
	return &immutableStorageClusterClient{cc}
}

func (c *immutableStorageClusterClient) PropagateWrite(ctx context.Context, in *PropagateWriteRequest, opts ...grpc.CallOption) (*PropagateWriteResponse, error) {
	out := new(PropagateWriteResponse)
	err := c.cc.Invoke(ctx, "/server.ImmutableStorageCluster/PropagateWrite", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *immutableStorageClusterClient) Sync(ctx context.Context, in *SyncRequest, opts ...grpc.CallOption) (*SyncResponse, error) {
	out := new(SyncResponse)
	err := c.cc.Invoke(ctx, "/server.ImmutableStorageCluster/Sync", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *immutableStorageClusterClient) GetNodetxtIPNS(ctx context.Context, in *GetNodetxtIPNSRequest, opts ...grpc.CallOption) (*GetNodetxtIPNSResponse, error) {
	out := new(GetNodetxtIPNSResponse)
	err := c.cc.Invoke(ctx, "/server.ImmutableStorageCluster/GetNodetxtIPNS", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ImmutableStorageClusterServer is the server API for ImmutableStorageCluster service.
// All implementations must embed UnimplementedImmutableStorageClusterServer
// for forward compatibility
type ImmutableStorageClusterServer interface {
	PropagateWrite(context.Context, *PropagateWriteRequest) (*PropagateWriteResponse, error)
	Sync(context.Context, *SyncRequest) (*SyncResponse, error)
	GetNodetxtIPNS(context.Context, *GetNodetxtIPNSRequest) (*GetNodetxtIPNSResponse, error)
	mustEmbedUnimplementedImmutableStorageClusterServer()
}

// UnimplementedImmutableStorageClusterServer must be embedded to have forward compatible implementations.
type UnimplementedImmutableStorageClusterServer struct {
}

func (UnimplementedImmutableStorageClusterServer) PropagateWrite(context.Context, *PropagateWriteRequest) (*PropagateWriteResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PropagateWrite not implemented")
}
func (UnimplementedImmutableStorageClusterServer) Sync(context.Context, *SyncRequest) (*SyncResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Sync not implemented")
}
func (UnimplementedImmutableStorageClusterServer) GetNodetxtIPNS(context.Context, *GetNodetxtIPNSRequest) (*GetNodetxtIPNSResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetNodetxtIPNS not implemented")
}
func (UnimplementedImmutableStorageClusterServer) mustEmbedUnimplementedImmutableStorageClusterServer() {
}

// UnsafeImmutableStorageClusterServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ImmutableStorageClusterServer will
// result in compilation errors.
type UnsafeImmutableStorageClusterServer interface {
	mustEmbedUnimplementedImmutableStorageClusterServer()
}

func RegisterImmutableStorageClusterServer(s grpc.ServiceRegistrar, srv ImmutableStorageClusterServer) {
	s.RegisterService(&ImmutableStorageCluster_ServiceDesc, srv)
}

func _ImmutableStorageCluster_PropagateWrite_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PropagateWriteRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ImmutableStorageClusterServer).PropagateWrite(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/server.ImmutableStorageCluster/PropagateWrite",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ImmutableStorageClusterServer).PropagateWrite(ctx, req.(*PropagateWriteRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ImmutableStorageCluster_Sync_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SyncRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ImmutableStorageClusterServer).Sync(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/server.ImmutableStorageCluster/Sync",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ImmutableStorageClusterServer).Sync(ctx, req.(*SyncRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ImmutableStorageCluster_GetNodetxtIPNS_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetNodetxtIPNSRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ImmutableStorageClusterServer).GetNodetxtIPNS(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/server.ImmutableStorageCluster/GetNodetxtIPNS",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ImmutableStorageClusterServer).GetNodetxtIPNS(ctx, req.(*GetNodetxtIPNSRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// ImmutableStorageCluster_ServiceDesc is the grpc.ServiceDesc for ImmutableStorageCluster service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ImmutableStorageCluster_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "server.ImmutableStorageCluster",
	HandlerType: (*ImmutableStorageClusterServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "PropagateWrite",
			Handler:    _ImmutableStorageCluster_PropagateWrite_Handler,
		},
		{
			MethodName: "Sync",
			Handler:    _ImmutableStorageCluster_Sync_Handler,
		},
		{
			MethodName: "GetNodetxtIPNS",
			Handler:    _ImmutableStorageCluster_GetNodetxtIPNS_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "protos/cluster.proto",
}