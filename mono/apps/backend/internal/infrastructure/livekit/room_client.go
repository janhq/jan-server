package livekit

import (
	"context"

	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"

	"jan-server/mono/apps/backend/internal/config"
)

// RoomClient provides access to LiveKit room management APIs.
type RoomClient struct {
	client *lksdk.RoomServiceClient
}

// NewRoomClient creates a new LiveKit room client.
func NewRoomClient(cfg *config.Config) *RoomClient {
	client := lksdk.NewRoomServiceClient(cfg.LiveKitWsURL, cfg.LiveKitAPIKey, cfg.LiveKitAPISecret)
	return &RoomClient{client: client}
}

// RoomInfo contains basic room information.
type RoomInfo struct {
	Name            string
	NumParticipants int
}

// ListActiveRooms returns all active rooms with participant counts.
func (c *RoomClient) ListActiveRooms(ctx context.Context) (map[string]RoomInfo, error) {
	resp, err := c.client.ListRooms(ctx, &livekit.ListRoomsRequest{})
	if err != nil {
		return nil, err
	}

	rooms := make(map[string]RoomInfo)
	for _, room := range resp.Rooms {
		rooms[room.Name] = RoomInfo{
			Name:            room.Name,
			NumParticipants: int(room.NumParticipants),
		}
	}
	return rooms, nil
}

// ListParticipants returns participant identities for a room.
func (c *RoomClient) ListParticipants(ctx context.Context, room string) ([]string, error) {
	resp, err := c.client.ListParticipants(ctx, &livekit.ListParticipantsRequest{
		Room: room,
	})
	if err != nil {
		return nil, err
	}

	identities := make([]string, 0, len(resp.Participants))
	for _, p := range resp.Participants {
		identities = append(identities, p.Identity)
	}
	return identities, nil
}
