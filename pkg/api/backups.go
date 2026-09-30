package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/m-vinc/maco/pkg/engine"
	"github.com/m-vinc/maco/pkg/jobs"
)

// @Summary listBackups
// @ID listBackups
// @Description List backups for a VM ID from listVMs. Use a returned timestamp unchanged with restoreBackup or deleteBackup.
// @x-maco {"expose":true,"readOnly":true,"transport":"json"}
// @Tags vms
// @Security BearerAuth
// @Produce json
// @Param id path string true "id"
// @Success 200 {array} engine.BackupInfo
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/vms/{id}/backups [get]
func (s *Server) listBackups(w http.ResponseWriter, r *http.Request) {
	backups, err := s.engine.ListBackups(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, backups)
}

// @Summary createBackup
// @ID createBackup
// @Description Create a backup of the selected VM. This queues a backup job; wait for success before treating the backup as usable, then inspect listBackups. Requires an administrator account.
// @x-maco {"expose":true,"readOnly":false,"transport":"json","wait":{"operationId":"waitJob","idField":"id","parameter":"id"}}
// @Tags vms
// @Security BearerAuth
// @Produce json
// @Param id path string true "id"
// @Success 202 {object} jobs.Job
// @Failure 401 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/vms/{id}/backups [post]
func (s *Server) createBackup(w http.ResponseWriter, r *http.Request) {
	s.submitJob(w, r, jobs.Payload{Action: "vm.backup", Target: chi.URLParam(r, "id")})
}

type restoreRequest struct {
	AsNew bool `json:"as_new" binding:"optional"`
}

// @Summary restoreBackup
// @ID restoreBackup
// @Description Restore the backup identified by VM id and timestamp from listBackups. Set body.as_new=true to restore as a new VM. False restores in place and can replace current VM data. Example body: {"as_new":true}. Wait for the restore job before using the VM. Requires an administrator account.
// @x-maco {"expose":true,"readOnly":false,"transport":"json","wait":{"operationId":"waitJob","idField":"id","parameter":"id"}}
// @Tags vms
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Param timestamp path string true "timestamp"
// @Param body body restoreRequest true "Request"
// @Success 202 {object} jobs.Job
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/vms/{id}/backups/{timestamp}/restore [post]
func (s *Server) restoreBackup(w http.ResponseWriter, r *http.Request) {
	var req restoreRequest
	if r.ContentLength != 0 {
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid restore settings")
			return
		}
	}

	s.submitJob(w, r, jobs.Payload{
		Action: "vm.backup.restore",
		Target: chi.URLParam(r, "id"),
		Backup: engine.BackupParams{Timestamp: chi.URLParam(r, "timestamp"), AsNew: req.AsNew},
	})
}

// @Summary getBackupSchedule
// @ID getBackupSchedule
// @Description Read the backup schedule and retention settings for a VM. Use the returned values to prepare a schedule update.
// @x-maco {"expose":true,"readOnly":true,"transport":"json"}
// @Tags vms
// @Security BearerAuth
// @Produce json
// @Param id path string true "id"
// @Success 200 {object} types.BackupSchedule
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/vms/{id}/backups/schedule [get]
func (s *Server) getBackupSchedule(w http.ResponseWriter, r *http.Request) {
	schedule, err := s.engine.GetSchedule(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, schedule)
}

// @Summary setBackupSchedule
// @ID setBackupSchedule
// @Description Set backup scheduling and retention for a VM. interval_hours is in hours, keep_last counts retained backups, and max_age_days is in days. Retention can delete old backups; review existing settings first. Requires an administrator account.
// @x-maco {"expose":true,"readOnly":false,"transport":"json"}
// @Tags vms
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Param body body engine.ScheduleParams true "Request"
// @Success 200 {object} types.BackupSchedule
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/vms/{id}/backups/schedule [put]
func (s *Server) setBackupSchedule(w http.ResponseWriter, r *http.Request) {
	var params engine.ScheduleParams
	if err := readJSON(r, &params); err != nil {
		writeError(w, http.StatusBadRequest, "invalid schedule settings")
		return
	}

	schedule, err := s.engine.SetSchedule(r.Context(), chi.URLParam(r, "id"), params)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, schedule)
}

// @Summary deleteBackup
// @ID deleteBackup
// @Description Permanently delete the backup selected by VM id and timestamp from listBackups. This does not delete the VM itself. Requires an administrator account.
// @x-maco {"expose":true,"readOnly":false,"transport":"json","wait":{"operationId":"waitJob","idField":"id","parameter":"id"}}
// @Tags vms
// @Security BearerAuth
// @Produce json
// @Param id path string true "id"
// @Param timestamp path string true "timestamp"
// @Success 202 {object} jobs.Job
// @Failure 401 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/vms/{id}/backups/{timestamp} [delete]
func (s *Server) deleteBackup(w http.ResponseWriter, r *http.Request) {
	s.submitJob(w, r, jobs.Payload{
		Action: "vm.backup.delete",
		Target: chi.URLParam(r, "id"),
		Backup: engine.BackupParams{Timestamp: chi.URLParam(r, "timestamp")},
	})
}
