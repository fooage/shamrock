package http_api

// ObjectMeta store object basic info and chunk list.
type ObjectMeta struct {
	Name      string   `json:"name"`
	Size      uint64   `json:"size"`
	ChunkList []string `json:"chunk_list"`
}

// ChunkMeta store chunk route info for client use.
type ChunkMeta struct {
	Parent     string `json:"parent"`
	Hash       string `json:"hash"`
	Index      int    `json:"index"`
	StoreGroup int    `json:"store_group"`
}
