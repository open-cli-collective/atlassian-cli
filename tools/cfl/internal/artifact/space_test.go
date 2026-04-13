package artifact

import (
	"testing"

	"github.com/open-cli-collective/atlassian-go/artifact"
	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/confluence-cli/api"
)

func TestProjectSpace(t *testing.T) {
	t.Parallel()

	space := &api.Space{
		ID:     "SPACE123",
		Key:    "DEV",
		Name:   "Development",
		Type:   "global",
		Status: "current",
		Description: &api.SpaceDescription{
			Plain: &api.DescriptionValue{
				Value: "Development space for testing",
			},
		},
	}

	t.Run("agent mode includes core fields only", func(t *testing.T) {
		art := ProjectSpace(space, artifact.Agent)

		testutil.Equal(t, "SPACE123", art.ID)
		testutil.Equal(t, "DEV", art.Key)
		testutil.Equal(t, "Development", art.Name)
		testutil.Equal(t, "", art.Type)        // not set in agent mode
		testutil.Equal(t, "", art.Status)      // not set in agent mode
		testutil.Equal(t, "", art.Description) // not set in agent mode
	})

	t.Run("full mode includes type, status, description", func(t *testing.T) {
		art := ProjectSpace(space, artifact.Full)

		testutil.Equal(t, "SPACE123", art.ID)
		testutil.Equal(t, "DEV", art.Key)
		testutil.Equal(t, "Development", art.Name)
		testutil.Equal(t, "global", art.Type)
		testutil.Equal(t, "current", art.Status)
		testutil.Equal(t, "Development space for testing", art.Description)
	})

	t.Run("handles nil description", func(t *testing.T) {
		spaceNoDesc := &api.Space{
			ID:   "SPACE123",
			Key:  "DEV",
			Name: "Development",
		}

		art := ProjectSpace(spaceNoDesc, artifact.Full)
		testutil.Equal(t, "", art.Description)
	})

	t.Run("handles description with nil plain", func(t *testing.T) {
		spaceNilPlain := &api.Space{
			ID:          "SPACE123",
			Key:         "DEV",
			Name:        "Development",
			Description: &api.SpaceDescription{},
		}

		art := ProjectSpace(spaceNilPlain, artifact.Full)
		testutil.Equal(t, "", art.Description)
	})
}

func TestProjectSpaces(t *testing.T) {
	t.Parallel()

	spaces := []api.Space{
		{ID: "1", Key: "DEV", Name: "Development"},
		{ID: "2", Key: "PROD", Name: "Production"},
	}

	arts := ProjectSpaces(spaces, artifact.Agent)

	testutil.Equal(t, 2, len(arts))
	testutil.Equal(t, "1", arts[0].ID)
	testutil.Equal(t, "DEV", arts[0].Key)
	testutil.Equal(t, "2", arts[1].ID)
	testutil.Equal(t, "PROD", arts[1].Key)
}
