package hub

type dataHub struct {
}

var _ DataHub = (*dataHub)(nil)

func (hub *dataHub) CreateStream(streamID string) (DataStreamInput, error) {
	return nil, nil
}

func (hub *dataHub) GetStreamOutput(streamID string) (DataStreamOutput, error) {
	return nil, nil
}
