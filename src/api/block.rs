use crate::block::{block_server::Block, *};
use core::panic;
use tonic::{Request, Response, Status};

#[derive(Debug, Default)]
pub struct BlockService {}

#[tonic::async_trait]
impl Block for BlockService {
    async fn fetch_block(
        &self,
        request: Request<FetchBlockReq>,
    ) -> Result<Response<FetchBlockResp>, Status> {
        panic!("implement me")
    }

    async fn store_block(
        &self,
        request: Request<StoreBlockReq>,
    ) -> Result<Response<StoreBlockResp>, Status> {
        panic!("implement me")
    }

    async fn delete_block(
        &self,
        request: Request<DeleteBlockReq>,
    ) -> Result<Response<DeleteBlockResp>, Status> {
        panic!("implement me")
    }
}
