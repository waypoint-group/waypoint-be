package service

// ChannelService provides business operations for channels.
type ChannelService struct {
	store ChannelStore
}

type ChannelStore interface{}

func NewChannelService(store ChannelStore) *ChannelService {
	return &ChannelService{store: store}
}
