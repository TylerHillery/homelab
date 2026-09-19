package hlims

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/TylerHillery/homelab/services/hlims/generated/api"
	database "github.com/TylerHillery/homelab/services/hlims/generated/db"
)

func (s apiServer) ListMachineUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := s.queries.ListMachineUsers(r.Context())
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	items := make([]api.MachineUser, 0, len(rows))
	for _, row := range rows {
		items = append(items, machineUserResponse(row.PublicID, row.MachinePublicID, row.Username, row.IsPreferred, row.Notes))
	}
	writeJSON(w, http.StatusOK, struct {
		Items []api.MachineUser `json:"items"`
	}{items})
}

type machineUserWriteValues struct {
	machine   database.GetMachineByPublicIDRow
	username  string
	preferred int64
	notes     sql.NullString
}

func (s apiServer) machineUserWriteValues(r *http.Request, body api.MachineUserWrite) (machineUserWriteValues, error) {
	machine, err := s.queries.GetMachineByPublicID(r.Context(), body.MachinePublicId)
	if err != nil {
		return machineUserWriteValues{}, err
	}
	username, err := normalizeMachineUsername(body.Username)
	if err != nil {
		return machineUserWriteValues{}, err
	}
	return machineUserWriteValues{machine: machine, username: username, preferred: boolInt(body.IsPreferred), notes: nullString(body.Notes, false)}, nil
}

func (s apiServer) CreateMachineUser(w http.ResponseWriter, r *http.Request) {
	body, ok := readWriteBody[api.MachineUserWrite](w, r)
	if !ok {
		return
	}
	values, err := s.machineUserWriteValues(r, body)
	if err != nil {
		s.writeMachineUserInputError(w, err)
		return
	}
	id, publicID, err := newIDs()
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	if err := s.saveMachineUser(r, values, id, publicID, false); err != nil {
		writeDatabaseError(w, err)
		return
	}
	s.writeMachineUser(w, r, publicID, http.StatusCreated)
}

func (s apiServer) GetMachineUser(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	s.writeMachineUser(w, r, publicID, http.StatusOK)
}

func (s apiServer) writeMachineUser(w http.ResponseWriter, r *http.Request, publicID string, status int) {
	row, err := s.queries.GetMachineUserByPublicID(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	writeJSON(w, status, machineUserResponse(row.PublicID, row.MachinePublicID, row.Username, row.IsPreferred, row.Notes))
}

func machineUserResponse(publicID, machinePublicID, username string, preferred int64, notes sql.NullString) api.MachineUser {
	return api.MachineUser{PublicId: publicID, MachinePublicId: machinePublicID, Username: username, IsPreferred: preferred == 1, Notes: stringPointer(notes)}
}

func (s apiServer) UpdateMachineUser(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	body, ok := readWriteBody[api.MachineUserWrite](w, r)
	if !ok {
		return
	}
	if _, err := s.queries.GetMachineUserByPublicID(r.Context(), publicID); err != nil {
		writeDatabaseError(w, err)
		return
	}
	values, err := s.machineUserWriteValues(r, body)
	if err != nil {
		s.writeMachineUserInputError(w, err)
		return
	}
	if err := s.saveMachineUser(r, values, "", publicID, true); err != nil {
		writeDatabaseError(w, err)
		return
	}
	s.writeMachineUser(w, r, publicID, http.StatusOK)
}

func (s apiServer) saveMachineUser(r *http.Request, values machineUserWriteValues, id, publicID string, update bool) error {
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	queries := s.queries.WithTx(tx)
	if values.preferred == 1 {
		if update {
			err = queries.ClearPreferredMachineUsers(r.Context(), database.ClearPreferredMachineUsersParams{MachineID: values.machine.ID, PublicID: publicID})
		} else {
			err = queries.ClearAllPreferredMachineUsers(r.Context(), values.machine.ID)
		}
		if err != nil {
			return err
		}
	}
	if update {
		_, err = queries.UpdateMachineUser(r.Context(), database.UpdateMachineUserParams{MachineID: values.machine.ID, Username: values.username, IsPreferred: values.preferred, Notes: values.notes, PublicID: publicID})
	} else {
		_, err = queries.CreateMachineUser(r.Context(), database.CreateMachineUserParams{ID: id, PublicID: publicID, MachineID: values.machine.ID, Username: values.username, IsPreferred: values.preferred, Notes: values.notes})
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s apiServer) writeMachineUserInputError(w http.ResponseWriter, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		writeDatabaseError(w, err)
		return
	}
	writeInputError(w, err)
}

func (s apiServer) DeleteMachineUser(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	rows, err := s.queries.DeleteMachineUser(r.Context(), publicID)
	deleteResource(w, rows, err)
}
