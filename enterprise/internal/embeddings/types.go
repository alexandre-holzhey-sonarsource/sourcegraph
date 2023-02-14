package embeddings

import "github.com/sourcegraph/sourcegraph/internal/api"

type EmbeddingIndex[T any] struct {
	Embeddings      []float32 `json:"embeddings"`
	ColumnDimension int       `json:"columnDimension"`
	RowMetadata     []T       `json:"rowMetadata"`
}

type RepoEmbeddingRowMetadata struct {
	FileName  string `json:"fileName"`
	StartLine int    `json:"startLine"`
	EndLine   int    `json:"endLine"`
}

type RepoEmbeddingIndex struct {
	RepoName  api.RepoName                              `json:"repoName"`
	Revision  api.CommitID                              `json:"revision"`
	CodeIndex *EmbeddingIndex[RepoEmbeddingRowMetadata] `json:"codeIndex"`
	TextIndex *EmbeddingIndex[RepoEmbeddingRowMetadata] `json:"textIndex"`
}

type EmbeddingSearchResults struct {
	CodeResults []EmbeddingSearchResult `json:"codeResults"`
	TextResults []EmbeddingSearchResult `json:"textResults"`
}

type EmbeddingSearchResult struct {
	FilePath  string `json:"filePath"`
	StartLine int    `json:"startLine"`
	EndLine   int    `json:"endLine"`
	Content   string `json:"content"`
}
