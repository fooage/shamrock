// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.21.12
// source: meta_service.proto

package meta_service

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

// MetaServiceClient is the client API for MetaService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type MetaServiceClient interface {
	// Query or update meta information about storage objects.
	QueryObjectMeta(ctx context.Context, in *QueryObjectMetaReq, opts ...grpc.CallOption) (*QueryObjectMetaResp, error)
	UpdateObjectStatus(ctx context.Context, in *UpdateObjectStatusReq, opts ...grpc.CallOption) (*UpdateObjectStatusResp, error)
	// Query or update chunk meta and routing information.
	QueryChunkMeta(ctx context.Context, in *QueryChunkMetaReq, opts ...grpc.CallOption) (*QueryChunkMetaResp, error)
	UpdateChunkStatus(ctx context.Context, in *UpdateChunkStatusReq, opts ...grpc.CallOption) (*UpdateChunkStatusResp, error)
	// The registration object completes the route table construction.
	RegisterObject(ctx context.Context, in *RegisterObjectReq, opts ...grpc.CallOption) (*RegisterObjectResp, error)
	// Proxy the block service node address which matched parameters.
	QueryStorageAddress(ctx context.Context, in *QueryStorageAddressReq, opts ...grpc.CallOption) (*QueryStorageAddressResp, error)
}

type metaServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewMetaServiceClient(cc grpc.ClientConnInterface) MetaServiceClient {
	return &metaServiceClient{cc}
}

func (c *metaServiceClient) QueryObjectMeta(ctx context.Context, in *QueryObjectMetaReq, opts ...grpc.CallOption) (*QueryObjectMetaResp, error) {
	out := new(QueryObjectMetaResp)
	err := c.cc.Invoke(ctx, "/meta_service.MetaService/QueryObjectMeta", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *metaServiceClient) UpdateObjectStatus(ctx context.Context, in *UpdateObjectStatusReq, opts ...grpc.CallOption) (*UpdateObjectStatusResp, error) {
	out := new(UpdateObjectStatusResp)
	err := c.cc.Invoke(ctx, "/meta_service.MetaService/UpdateObjectStatus", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *metaServiceClient) QueryChunkMeta(ctx context.Context, in *QueryChunkMetaReq, opts ...grpc.CallOption) (*QueryChunkMetaResp, error) {
	out := new(QueryChunkMetaResp)
	err := c.cc.Invoke(ctx, "/meta_service.MetaService/QueryChunkMeta", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *metaServiceClient) UpdateChunkStatus(ctx context.Context, in *UpdateChunkStatusReq, opts ...grpc.CallOption) (*UpdateChunkStatusResp, error) {
	out := new(UpdateChunkStatusResp)
	err := c.cc.Invoke(ctx, "/meta_service.MetaService/UpdateChunkStatus", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *metaServiceClient) RegisterObject(ctx context.Context, in *RegisterObjectReq, opts ...grpc.CallOption) (*RegisterObjectResp, error) {
	out := new(RegisterObjectResp)
	err := c.cc.Invoke(ctx, "/meta_service.MetaService/RegisterObject", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *metaServiceClient) QueryStorageAddress(ctx context.Context, in *QueryStorageAddressReq, opts ...grpc.CallOption) (*QueryStorageAddressResp, error) {
	out := new(QueryStorageAddressResp)
	err := c.cc.Invoke(ctx, "/meta_service.MetaService/QueryStorageAddress", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MetaServiceServer is the server API for MetaService service.
// All implementations must embed UnimplementedMetaServiceServer
// for forward compatibility
type MetaServiceServer interface {
	// Query or update meta information about storage objects.
	QueryObjectMeta(context.Context, *QueryObjectMetaReq) (*QueryObjectMetaResp, error)
	UpdateObjectStatus(context.Context, *UpdateObjectStatusReq) (*UpdateObjectStatusResp, error)
	// Query or update chunk meta and routing information.
	QueryChunkMeta(context.Context, *QueryChunkMetaReq) (*QueryChunkMetaResp, error)
	UpdateChunkStatus(context.Context, *UpdateChunkStatusReq) (*UpdateChunkStatusResp, error)
	// The registration object completes the route table construction.
	RegisterObject(context.Context, *RegisterObjectReq) (*RegisterObjectResp, error)
	// Proxy the block service node address which matched parameters.
	QueryStorageAddress(context.Context, *QueryStorageAddressReq) (*QueryStorageAddressResp, error)
	mustEmbedUnimplementedMetaServiceServer()
}

// UnimplementedMetaServiceServer must be embedded to have forward compatible implementations.
type UnimplementedMetaServiceServer struct {
}

func (UnimplementedMetaServiceServer) QueryObjectMeta(context.Context, *QueryObjectMetaReq) (*QueryObjectMetaResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method QueryObjectMeta not implemented")
}
func (UnimplementedMetaServiceServer) UpdateObjectStatus(context.Context, *UpdateObjectStatusReq) (*UpdateObjectStatusResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateObjectStatus not implemented")
}
func (UnimplementedMetaServiceServer) QueryChunkMeta(context.Context, *QueryChunkMetaReq) (*QueryChunkMetaResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method QueryChunkMeta not implemented")
}
func (UnimplementedMetaServiceServer) UpdateChunkStatus(context.Context, *UpdateChunkStatusReq) (*UpdateChunkStatusResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateChunkStatus not implemented")
}
func (UnimplementedMetaServiceServer) RegisterObject(context.Context, *RegisterObjectReq) (*RegisterObjectResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RegisterObject not implemented")
}
func (UnimplementedMetaServiceServer) QueryStorageAddress(context.Context, *QueryStorageAddressReq) (*QueryStorageAddressResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method QueryStorageAddress not implemented")
}
func (UnimplementedMetaServiceServer) mustEmbedUnimplementedMetaServiceServer() {}

// UnsafeMetaServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to MetaServiceServer will
// result in compilation errors.
type UnsafeMetaServiceServer interface {
	mustEmbedUnimplementedMetaServiceServer()
}

func RegisterMetaServiceServer(s grpc.ServiceRegistrar, srv MetaServiceServer) {
	s.RegisterService(&MetaService_ServiceDesc, srv)
}

func _MetaService_QueryObjectMeta_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryObjectMetaReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MetaServiceServer).QueryObjectMeta(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/meta_service.MetaService/QueryObjectMeta",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MetaServiceServer).QueryObjectMeta(ctx, req.(*QueryObjectMetaReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _MetaService_UpdateObjectStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateObjectStatusReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MetaServiceServer).UpdateObjectStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/meta_service.MetaService/UpdateObjectStatus",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MetaServiceServer).UpdateObjectStatus(ctx, req.(*UpdateObjectStatusReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _MetaService_QueryChunkMeta_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryChunkMetaReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MetaServiceServer).QueryChunkMeta(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/meta_service.MetaService/QueryChunkMeta",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MetaServiceServer).QueryChunkMeta(ctx, req.(*QueryChunkMetaReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _MetaService_UpdateChunkStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateChunkStatusReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MetaServiceServer).UpdateChunkStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/meta_service.MetaService/UpdateChunkStatus",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MetaServiceServer).UpdateChunkStatus(ctx, req.(*UpdateChunkStatusReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _MetaService_RegisterObject_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RegisterObjectReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MetaServiceServer).RegisterObject(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/meta_service.MetaService/RegisterObject",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MetaServiceServer).RegisterObject(ctx, req.(*RegisterObjectReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _MetaService_QueryStorageAddress_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryStorageAddressReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MetaServiceServer).QueryStorageAddress(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/meta_service.MetaService/QueryStorageAddress",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MetaServiceServer).QueryStorageAddress(ctx, req.(*QueryStorageAddressReq))
	}
	return interceptor(ctx, in, info, handler)
}

// MetaService_ServiceDesc is the grpc.ServiceDesc for MetaService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var MetaService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "meta_service.MetaService",
	HandlerType: (*MetaServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "QueryObjectMeta",
			Handler:    _MetaService_QueryObjectMeta_Handler,
		},
		{
			MethodName: "UpdateObjectStatus",
			Handler:    _MetaService_UpdateObjectStatus_Handler,
		},
		{
			MethodName: "QueryChunkMeta",
			Handler:    _MetaService_QueryChunkMeta_Handler,
		},
		{
			MethodName: "UpdateChunkStatus",
			Handler:    _MetaService_UpdateChunkStatus_Handler,
		},
		{
			MethodName: "RegisterObject",
			Handler:    _MetaService_RegisterObject_Handler,
		},
		{
			MethodName: "QueryStorageAddress",
			Handler:    _MetaService_QueryStorageAddress_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "meta_service.proto",
}
