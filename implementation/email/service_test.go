package email

import (
	"testing"

	// "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmailValidation(t *testing.T) {
	testCases := []struct {
		desc       string
		e          Entity
		shouldFail bool
		err        error
	}{
		{
			desc: "should pass",
			e: Entity{
				FromName: "xyz",
				ToList: []NameAddr{
					{
						EmailAddr: "xyz",
						UserName:  "xyz",
					},
				},
				Subject: "test",
				Body:    "its testing time !!!",
			},
			shouldFail: false,
			err:        nil,
		},
		{
			desc: "should return empty email address error",
			e: Entity{
				FromName: "xyz",
				ToList: []NameAddr{
					{
						EmailAddr: "",
						UserName:  "xyz",
					},
				},
				Subject: "test",
				Body:    "its testing time !!!",
			},
			shouldFail: true,
			err:        ErrEmptyAddr,
		},
		{
			desc: "should return empty toList error",
			e: Entity{
				FromName: "xyz",
				Subject:  "test",
				Body:     "its testing time !!!",
			},
			shouldFail: true,
			err:        ErrEmptyToList,
		},
		{
			desc: "passing empty tolist",
			e: Entity{
				FromName: "xyz",
				ToList:   []NameAddr{},
				Subject:  "test",
				Body:     "its testing time !!!",
			},
			shouldFail: true,
			err:        ErrEmptyToList,
		},
		{
			desc: "passing empty tolist2",
			e: Entity{
				FromName: "xyz",
				ToList: []NameAddr{
					{},
				},
				Subject: "test",
				Body:    "its testing time !!!",
			},
			shouldFail: true,
			err:        ErrEmptyAddr,
		},
	}

	for i := range testCases {
		t.Run(testCases[i].desc, func(t *testing.T) {
			err := testCases[i].e.ToListValidation()
			if !testCases[i].shouldFail {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, testCases[i].err, "return err didnot match with the expected err")
			}
		})
	}
}
