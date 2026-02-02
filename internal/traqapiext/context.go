package traqapiext

import (
	"context"
	"errors"
	"fmt"
	"image"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/motoki317/sc"
	"github.com/ras0q/lazytraq/internal/traqapi"
	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers/rasterizer"
)

type Context struct {
	apiHost string
	client  *traqapi.Client

	Messages    *sc.Cache[uuid.UUID, []traqapi.Message]
	Users       *sc.Cache[struct{}, []traqapi.User]
	Stamps      *sc.Cache[struct{}, []traqapi.StampWithThumbnail]
	StampImages *sc.Cache[uuid.UUID, image.Image]
	Channels    *sc.Cache[struct{}, *traqapi.ChannelList]
	Me          *sc.Cache[struct{}, *traqapi.MyUserDetail]
}

func NewContext(apiHost string, securitySource *SecuritySource) (*Context, error) {
	c := new(Context)
	if err := c.SwitchHost(apiHost, securitySource); err != nil {
		return nil, fmt.Errorf("switch host: %w", err)
	}

	return c, nil
}

func (c *Context) SwitchHost(apiHost string, securitySource *SecuritySource) error {
	httpClient := http.DefaultClient
	httpClient.Timeout = 10 * time.Second
	traqClient, err := traqapi.NewClient(
		fmt.Sprintf("https://%s/api/v3", apiHost),
		securitySource,
		traqapi.WithClient(httpClient),
	)
	if err != nil {
		return fmt.Errorf("create traq client: %w", err)
	}

	c.apiHost = apiHost
	c.client = traqClient

	c.Messages, err = newMessagesStore(traqClient)
	if err != nil {
		return fmt.Errorf("create messages store: %w", err)
	}

	c.Users, err = newUsersStore(traqClient)
	if err != nil {
		return fmt.Errorf("create users store: %w", err)
	}

	c.Stamps, err = newStampsStore(traqClient)
	if err != nil {
		return fmt.Errorf("create stamps store: %w", err)
	}

	c.StampImages, err = newStampImagesStore(traqClient)
	if err != nil {
		return fmt.Errorf("create stamp images store: %w", err)
	}

	c.Channels, err = newChannelsStore(traqClient)
	if err != nil {
		return fmt.Errorf("create channels store: %w", err)
	}

	c.Me, err = newMeStore(traqClient)
	if err != nil {
		return fmt.Errorf("create me store: %w", err)
	}

	return nil
}

func (c *Context) PostMessage(ctx context.Context, request traqapi.PostMessageRequest, channelID uuid.UUID) (traqapi.PostMessageRes, error) {
	res, err := c.client.PostMessage(
		ctx,
		traqapi.NewOptPostMessageRequest(request),
		traqapi.PostMessageParams{
			ChannelId: channelID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("post to channel %s: %w", channelID, err)
	}

	c.Messages.Forget(channelID)

	return res, nil
}

func (c *Context) GetStampImage(ctx context.Context, stampID uuid.UUID) (traqapi.GetStampImageRes, error) {
	return c.client.GetStampImage(ctx, traqapi.GetStampImageParams{
		StampId: stampID,
	})
}

func wrapf(errp *error, format string, args ...any) {
	if *errp != nil {
		*errp = fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), *errp)
	}
}

func newMessagesStore(traqClient *traqapi.Client) (*sc.Cache[uuid.UUID, []traqapi.Message], error) {
	freshFor := time.Minute * 5
	ttl := time.Minute * 10

	return sc.New(func(ctx context.Context, channelID uuid.UUID) (messages []traqapi.Message, err error) {
		defer wrapf(&err, "get messages from traQ for channel %s", channelID.String())

		res, err := traqClient.GetMessages(ctx, traqapi.GetMessagesParams{
			ChannelId: channelID,
		})
		if err != nil {
			return nil, err
		}

		switch res := res.(type) {
		case *traqapi.GetMessagesOKHeaders:
			return res.Response, nil

		case *traqapi.GetMessagesBadRequest:
			return nil, errors.New("bad request")

		case *traqapi.GetMessagesNotFound:
			return nil, errors.New("not found")

		default:
			return nil, fmt.Errorf("unreachable error")
		}
	}, freshFor, ttl)
}

func newUsersStore(traqClient *traqapi.Client) (*sc.Cache[struct{}, []traqapi.User], error) {
	freshFor := time.Minute * 5
	ttl := time.Minute * 10

	return sc.New(func(ctx context.Context, _ struct{}) (users []traqapi.User, err error) {
		defer wrapf(&err, "get users from traQ")

		res, err := traqClient.GetUsers(ctx, traqapi.GetUsersParams{})
		if err != nil {
			return nil, err
		}

		switch res := res.(type) {
		case *traqapi.GetUsersOKApplicationJSON:
			return *res, nil

		case *traqapi.GetUsersBadRequest:
			return nil, errors.New("bad request")

		default:
			return nil, fmt.Errorf("unreachable error")
		}
	}, freshFor, ttl)
}

func newStampsStore(traqClient *traqapi.Client) (*sc.Cache[struct{}, []traqapi.StampWithThumbnail], error) {
	freshFor := time.Minute * 5
	ttl := time.Minute * 10

	return sc.New(func(ctx context.Context, _ struct{}) (stamps []traqapi.StampWithThumbnail, err error) {
		defer wrapf(&err, "get stamps from traQ")

		stamps, err = traqClient.GetStamps(ctx, traqapi.GetStampsParams{
			IncludeUnicode: traqapi.NewOptBool(true),
		})
		if err != nil {
			return nil, err
		}

		return stamps, nil
	}, freshFor, ttl)
}

func newStampImagesStore(traqClient *traqapi.Client) (*sc.Cache[uuid.UUID, image.Image], error) {
	freshFor := time.Minute * 5
	ttl := time.Minute * 10

	return sc.New(func(ctx context.Context, stampID uuid.UUID) (image.Image, error) {
		res, err := traqClient.GetStampImage(ctx, traqapi.GetStampImageParams{
			StampId: stampID,
		})
		if err != nil {
			return nil, fmt.Errorf("get stamp image from traQ: %w", err)
		}

		var img image.Image
		switch res := res.(type) {
		case *traqapi.GetStampImageNotFound:
			return nil, fmt.Errorf("get stamp image from traQ: not found")

		case *traqapi.GetStampImageOKImageGIF:
			img, _, err = image.Decode(res.Data)
			if err != nil {
				return nil, fmt.Errorf("decode file to image: %w", err)
			}

		case *traqapi.GetStampImageOKImageJpeg:
			img, _, err = image.Decode(res.Data)
			if err != nil {
				return nil, fmt.Errorf("decode file to image: %w", err)
			}

		case *traqapi.GetStampImageOKImagePNG:
			img, _, err = image.Decode(res.Data)
			if err != nil {
				return nil, fmt.Errorf("decode file to image: %w", err)
			}

		case *traqapi.GetStampImageOKImageSvgXML:
			c, err := canvas.ParseSVG(res.Data)
			if err != nil {
				return nil, fmt.Errorf("parse svg: %w", err)
			}

			img = rasterizer.Draw(c, 96.0, canvas.DefaultColorSpace)
		}

		return img, nil
	}, freshFor, ttl)
}

func newChannelsStore(traqClient *traqapi.Client) (*sc.Cache[struct{}, *traqapi.ChannelList], error) {
	freshFor := time.Minute * 5
	ttl := time.Minute * 10

	return sc.New(func(ctx context.Context, _ struct{}) (channels *traqapi.ChannelList, err error) {
		defer wrapf(&err, "get channels from traQ")

		channels, err = traqClient.GetChannels(ctx, traqapi.GetChannelsParams{})
		if err != nil {
			return nil, err
		}

		return channels, nil
	}, freshFor, ttl)
}

func newMeStore(traqClient *traqapi.Client) (*sc.Cache[struct{}, *traqapi.MyUserDetail], error) {
	freshFor := time.Minute * 5
	ttl := time.Minute * 10

	return sc.New(func(ctx context.Context, _ struct{}) (me *traqapi.MyUserDetail, err error) {
		defer wrapf(&err, "get me from traQ")

		me, err = traqClient.GetMe(ctx)
		if err != nil {
			return nil, err
		}

		return me, nil
	}, freshFor, ttl)
}
