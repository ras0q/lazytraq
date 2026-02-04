package traqapiext

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/google/uuid"
	"github.com/motoki317/sc"
	"github.com/ras0q/goalie"
	"github.com/ras0q/lazytraq/internal/traqapi"
	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers/rasterizer"
)

func init() {
	// TODO: https://github.com/tdewolff/canvas/issues/372
	image.RegisterFormat("svg", "<svg", func(r io.Reader) (image.Image, error) {
		c, err := canvas.ParseSVG(r)
		if err != nil {
			return nil, fmt.Errorf("parse svg: %w", err)
		}

		img := rasterizer.Draw(c, 96.0, canvas.DefaultColorSpace)
		return img, nil
	}, func(r io.Reader) (image.Config, error) {
		c, err := canvas.ParseSVG(r)
		if err != nil {
			return image.Config{}, fmt.Errorf("parse svg: %w", err)
		}

		return image.Config{
			ColorModel: color.RGBAModel,
			Width:      int(c.W),
			Height:     int(c.H),
		}, nil
	})
}

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

func newStampImagesStore(traqClient *traqapi.Client) (_ *sc.Cache[uuid.UUID, image.Image], err error) {
	g := goalie.New()
	defer g.Collect(&err)

	freshFor := time.Minute * 5
	ttl := time.Minute * 10

	return sc.New(func(ctx context.Context, stampID uuid.UUID) (image.Image, error) {
		baseCacheDir, err := os.UserCacheDir()
		if err != nil {
			return nil, fmt.Errorf("get user cache dir: %w", err)
		}

		cacheDir := path.Join(baseCacheDir, "lazytraq", "stamp_images")
		if err := os.MkdirAll(cacheDir, os.ModePerm); err != nil {
			return nil, fmt.Errorf("create stamp image cache dir: %w", err)
		}

		cacheRoot, err := os.OpenRoot(cacheDir)
		if err != nil {
			return nil, fmt.Errorf("open stamp image cache dir: %w", err)
		}
		defer g.Guard(cacheRoot.Close)

		stampCacheDir := path.Join(cacheDir, stampID.String())
		entries, err := os.ReadDir(stampCacheDir)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("read stamp image cache subdir (%s): %w", stampCacheDir, err)
		}

		if len(entries) > 0 {
			filename := entries[0].Name()
			f, err := os.Open(path.Join(stampCacheDir, filename))
			if err == nil {
				defer g.Guard(f.Close)

				img, _, err := image.Decode(f)
				if err != nil {
					return nil, fmt.Errorf("decode cached stamp image (%s): %w", filename, err)
				}

				return img, nil
			}
		}

		res, err := traqClient.GetStampImage(ctx, traqapi.GetStampImageParams{
			StampId: stampID,
		})
		if err != nil {
			return nil, fmt.Errorf("get stamp image from traQ: %w", err)
		}

		var r io.Reader
		var ext string
		switch res := res.(type) {
		case *traqapi.GetStampImageNotFound:
			return nil, fmt.Errorf("stamp image not found")
		case *traqapi.GetStampImageOKImageGIF:
			r = res
			ext = "gif"
		case *traqapi.GetStampImageOKImageJpeg:
			r = res
			ext = "jpeg"
		case *traqapi.GetStampImageOKImagePNG:
			r = res
			ext = "png"
		case *traqapi.GetStampImageOKImageSvgXML:
			r = res
			ext = "svg"
		default:
			return nil, fmt.Errorf("unreachable error")
		}

		b, err := io.ReadAll(r)
		if err != nil {
			return nil, fmt.Errorf("read stamp image: %w", err)
		}

		img, _, err := image.Decode(bytes.NewReader(b))
		if err != nil {
			return nil, fmt.Errorf("decode stamp image: %w", err)
		}

		if err := cacheRoot.MkdirAll(stampID.String(), os.ModePerm); err != nil {
			return nil, fmt.Errorf("create stamp image cache subdir (%s): %w", stampID.String(), err)
		}

		filename := fmt.Sprintf("%s/%s.%s", stampID, stampID, ext)
		f, err := cacheRoot.Create(filename)
		if err != nil {
			return nil, fmt.Errorf("create temp file (%s) for stamp image: %w", filename, err)
		}
		defer g.Guard(f.Close)

		_, err = io.Copy(f, bytes.NewReader(b))
		if err != nil {
			return nil, fmt.Errorf("copy stamp image data to temp file (%s): %w", filename, err)
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
