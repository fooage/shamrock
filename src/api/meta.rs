use crate::meta::{meta_server::Meta, *};
use core::panic;
use tonic::{Request, Response, Status};

#[derive(Debug, Default)]
pub struct MetaService {}

#[tonic::async_trait]
impl Meta for MetaService {
    async fn query_object_meta(
        &self,
        request: Request<QueryObjectMetaReq>,
    ) -> Result<Response<QueryObjectMetaResp>, Status> {
        panic!("implement me")
    }

    async fn query_object_keys(
        &self,
        request: Request<QueryObjectKeysReq>,
    ) -> Result<Response<QueryObjectKeysResp>, Status> {
        panic!("implement me")
    }

    async fn update_object_status(
        &self,
        request: Request<UpdateObjectStatusReq>,
    ) -> Result<Response<UpdateObjectStatusResp>, Status> {
        panic!("implement me")
    }

    async fn query_chunk_meta(
        &self,
        request: Request<QueryChunkMetaReq>,
    ) -> Result<Response<QueryChunkMetaResp>, Status> {
        panic!("implement me")
    }

    async fn update_chunk_status(
        &self,
        request: Request<UpdateChunkStatusReq>,
    ) -> Result<Response<UpdateChunkStatusResp>, Status> {
        panic!("implement me")
    }

    async fn register_object(
        &self,
        request: Request<RegisterObjectReq>,
    ) -> Result<Response<RegisterObjectResp>, Status> {
        panic!("implement me")
    }

    async fn query_storage_address(
        &self,
        request: Request<QueryStorageAddressReq>,
    ) -> Result<Response<QueryStorageAddressResp>, Status> {
        panic!("implement me")
    }
}
