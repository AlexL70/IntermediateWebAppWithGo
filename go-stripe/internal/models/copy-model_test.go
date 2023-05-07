package models

import (
	"testing"
	"time"
)

func Test_CopyUser(t *testing.T) {
	var currTime = time.Now()
	var theTests = []struct {
		src    User
		dest   User
		result User
	}{
		{src: User{
			DBEntity: DBEntity{
				ID:        444,
				CreatedAt: time.Date(1999, 9, 9, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(1999, 9, 9, 0, 0, 0, 0, time.UTC),
			},
			FirstName: "Alex",
			LastName:  "Smith",
			Password:  "new_usr_pwd",
			Email:     "alex@smith.com",
		},
			dest: User{
				DBEntity: DBEntity{
					ID:        12,
					CreatedAt: currTime,
					UpdatedAt: currTime,
				},
				FirstName: "John",
				LastName:  "Dow",
				Password:  "old_pwd",
				Email:     "john@dow.com",
			},
			result: User{
				DBEntity: DBEntity{
					ID:        12,
					CreatedAt: currTime,
					UpdatedAt: currTime,
				},
				FirstName: "Alex",
				LastName:  "Smith",
				Password:  "old_pwd",
				Email:     "alex@smith.com",
			},
		},
	}

	for _, e := range theTests {
		err := modelCopy(&e.dest, e.src)
		if err != nil {
			t.Errorf("error copying data: %s", err)
		}
		if e.dest.ID != e.result.ID {
			t.Errorf("bad copying: expected %v but got %v", e.result.ID, e.dest.ID)
		}
		if e.dest.FirstName != e.result.FirstName {
			t.Errorf("bad copying: expected %v but got %v", e.result.FirstName, e.dest.FirstName)
		}
		if e.dest.LastName != e.result.LastName {
			t.Errorf("bad copying: expected %v but got %v", e.result.LastName, e.dest.LastName)
		}
		if e.dest.Email != e.result.Email {
			t.Errorf("bad copying: expected %v but got %v", e.result.Email, e.dest.Email)
		}
		if e.dest.Password != e.result.Password {
			t.Errorf("bad copying: expected %v but got %v", e.result.Password, e.dest.Password)
		}
		if e.dest.CreatedAt != e.result.CreatedAt {
			t.Errorf("bad copying: expected %v but got %v", e.result.CreatedAt, e.dest.CreatedAt)
		}
		if e.dest.UpdatedAt != e.result.UpdatedAt {
			t.Errorf("bad copying: expected %v but got %v", e.result.UpdatedAt, e.dest.UpdatedAt)
		}
	}
}
