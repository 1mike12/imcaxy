package repositories

type CachedFile struct {
	ID string
	FileURL string
	Checksum string
	Params map[string]string
}

type CachedFileRepository struct {
	
}