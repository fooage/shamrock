// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.21.12
// source: block_service.proto

package block_service

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

// BlockServiceClient is the client API for BlockService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type BlockServiceClient interface {
	// To obtain object block according to unique key, can set whether to pull from the Master node.
	FetchBlock(ctx context.Context, in *FetchBlockReq, opts ...grpc.CallOption) (*FetchBlockResp, error)
	// Store the object block to the Block service group.
	StoreBlock(ctx context.Context, in *StoreBlockReq, opts ...grpc.CallOption) (*StoreBlockResp, error)
	// Delete object block by unique key.
	DeleteBlock(ctx context.Context, in *DeleteBlockReq, opts ...grpc.CallOption) (*DeleteBlockResp, error)
}

type blockServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewBlockServiceClient(cc grpc.ClientConnInterface) BlockServiceClient {
	return &blockServiceClient{cc}
}

func (c *blockServiceClient) FetchBlock(ctx context.Context, in *FetchBlockReq, opts ...grpc.CallOption) (*FetchBlockResp, error) {
	out := new(FetchBlockResp)
	err := c.cc.Invoke(ctx, "/block_service.BlockService/FetchBlock", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *blockServiceClient) StoreBlock(ctx context.Context, in *StoreBlockReq, opts ...grpc.CallOption) (*StoreBlockResp, error) {
	out := new(StoreBlockResp)
	err := c.cc.Invoke(ctx, "/block_service.BlockService/StoreBlock", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *blockServiceClient) DeleteBlock(ctx context.Context, in *DeleteBlockReq, opts ...grpc.CallOption) (*DeleteBlockResp, error) {
	out := new(DeleteBlockResp)
	err := c.cc.Invoke(ctx, "/block_service.BlockService/DeleteBlock", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// BlockServiceServer is the server API for BlockService service.
// All implementations must embed UnimplementedBlockServiceServer
// for forward compatibility
type BlockServiceServer interface {
	// To obtain object block according to unique key, can set whether to pull from the Master node.
	FetchBlock(context.Context, *FetchBlockReq) (*FetchBlockResp, error)
	// Store the object block to the Block service group.
	StoreBlock(context.Context, *StoreBlockReq) (*StoreBlockResp, error)
	// Delete object block by unique key.
	DeleteBlock(context.Context, *DeleteBlockReq) (*DeleteBlockResp, error)
	mustEmbedUnimplementedBlockServiceServer()
}

// UnimplementedBlockServiceServer must be embedded to have forward compatible implementations.
type UnimplementedBlockServiceServer struct {
}

func (UnimplementedBlockServiceServer) FetchBlock(context.Context, *FetchBlockReq) (*FetchBlockResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method FetchBlock not implemented")
}
func (UnimplementedBlockServiceServer) StoreBlock(context.Context, *StoreBlockReq) (*StoreBlockResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StoreBlock not implemented")
}
func (UnimplementedBlockServiceServer) DeleteBlock(context.Context, *DeleteBlockReq) (*DeleteBlockResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteBlock not implemented")
}
func (UnimplementedBlockServiceServer) mustEmbedUnimplementedBlockServiceServer() {}

// UnsafeBlockServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to BlockServiceServer will
// result in compilation errors.
type UnsafeBlockServiceServer interface {
	mustEmbedUnimplementedBlockServiceServer()
}

func RegisterBlockServiceServer(s grpc.ServiceRegistrar, srv BlockServiceServer) {
	s.RegisterService(&BlockService_ServiceDesc, srv)
}

func _BlockService_FetchBlock_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FetchBlockReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlockServiceServer).FetchBlock(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/block_service.BlockService/FetchBlock",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlockServiceServer).FetchBlock(ctx, req.(*FetchBlockReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _BlockService_StoreBlock_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StoreBlockReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlockServiceServer).StoreBlock(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/block_service.BlockService/StoreBlock",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlockServiceServer).StoreBlock(ctx, req.(*StoreBlockReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _BlockService_DeleteBlock_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteBlockReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlockServiceServer).DeleteBlock(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/block_service.BlockService/DeleteBlock",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlockServiceServer).DeleteBlock(ctx, req.(*DeleteBlockReq))
	}
	return interceptor(ctx, in, info, handler)
}

// BlockService_ServiceDesc is the grpc.ServiceDesc for BlockService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var BlockService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "block_service.BlockService",
	HandlerType: (*BlockServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "FetchBlock",
			Handler:    _BlockService_FetchBlock_Handler,
		},
		{
			MethodName: "StoreBlock",
			Handler:    _BlockService_StoreBlock_Handler,
		},
		{
			MethodName: "DeleteBlock",
			Handler:    _BlockService_DeleteBlock_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "block_service.proto",
}
