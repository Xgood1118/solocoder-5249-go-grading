package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"grading-system/model"
)

type Repository struct {
	mu              sync.RWMutex
	papers          map[string]model.Paper
	students        map[string]model.Student
	submissions     map[string]model.PaperSubmission
	exportTasks     map[string]model.ExportTask
	dataDir         string
	snapshotDir     string
}

var repo *Repository
var once sync.Once

func GetRepository(dataDir string) *Repository {
	once.Do(func() {
		repo = &Repository{
			papers:      make(map[string]model.Paper),
			students:    make(map[string]model.Student),
			submissions: make(map[string]model.PaperSubmission),
			exportTasks: make(map[string]model.ExportTask),
			dataDir:     dataDir,
			snapshotDir: filepath.Join(dataDir, "snapshots"),
		}
		os.MkdirAll(repo.snapshotDir, 0755)
	})
	return repo
}

func (r *Repository) LoadData() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	papersFile := filepath.Join(r.dataDir, "papers.json")
	if _, err := os.Stat(papersFile); err == nil {
		data, err := os.ReadFile(papersFile)
		if err != nil {
			return fmt.Errorf("read papers.json: %w", err)
		}
		var papers []model.Paper
		if err := json.Unmarshal(data, &papers); err != nil {
			return fmt.Errorf("parse papers.json: %w", err)
		}
		for _, p := range papers {
			r.papers[p.ID] = p
		}
	}

	studentsFile := filepath.Join(r.dataDir, "students.json")
	if _, err := os.Stat(studentsFile); err == nil {
		data, err := os.ReadFile(studentsFile)
		if err != nil {
			return fmt.Errorf("read students.json: %w", err)
		}
		var students []model.Student
		if err := json.Unmarshal(data, &students); err != nil {
			return fmt.Errorf("parse students.json: %w", err)
		}
		for _, s := range students {
			r.students[s.ID] = s
		}
	}

	submissionsFile := filepath.Join(r.dataDir, "submissions.json")
	if _, err := os.Stat(submissionsFile); err == nil {
		data, err := os.ReadFile(submissionsFile)
		if err != nil {
			return fmt.Errorf("read submissions.json: %w", err)
		}
		var submissions []model.PaperSubmission
		if err := json.Unmarshal(data, &submissions); err != nil {
			return fmt.Errorf("parse submissions.json: %w", err)
		}
		for _, s := range submissions {
			r.submissions[s.ID] = s
		}
	}

	return nil
}

func (r *Repository) ListPapers() []model.Paper {
	r.mu.RLock()
	defer r.mu.RUnlock()
	papers := make([]model.Paper, 0, len(r.papers))
	for _, p := range r.papers {
		papers = append(papers, p)
	}
	return papers
}

func (r *Repository) GetPaper(id string) (model.Paper, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.papers[id]
	return p, ok
}

func (r *Repository) ListStudents() []model.Student {
	r.mu.RLock()
	defer r.mu.RUnlock()
	students := make([]model.Student, 0, len(r.students))
	for _, s := range r.students {
		students = append(students, s)
	}
	return students
}

func (r *Repository) GetStudent(id string) (model.Student, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.students[id]
	return s, ok
}

func (r *Repository) ListSubmissions() []model.PaperSubmission {
	r.mu.RLock()
	defer r.mu.RUnlock()
	subs := make([]model.PaperSubmission, 0, len(r.submissions))
	for _, s := range r.submissions {
		subs = append(subs, s)
	}
	return subs
}

func (r *Repository) ListSubmissionsByPaper(paperID string) []model.PaperSubmission {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var subs []model.PaperSubmission
	for _, s := range r.submissions {
		if s.PaperID == paperID {
			subs = append(subs, s)
		}
	}
	return subs
}

func (r *Repository) GetSubmission(id string) (model.PaperSubmission, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.submissions[id]
	return s, ok
}

func (r *Repository) GetSubmissionByPaperAndStudent(paperID, studentID string) (model.PaperSubmission, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, s := range r.submissions {
		if s.PaperID == paperID && s.StudentID == studentID {
			return s, true
		}
	}
	return model.PaperSubmission{}, false
}

func (r *Repository) SaveSubmission(sub model.PaperSubmission) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.submissions[sub.ID] = sub
}

func (r *Repository) UpdateSubmission(sub model.PaperSubmission) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.submissions[sub.ID]; !ok {
		return false
	}
	r.submissions[sub.ID] = sub
	return true
}

func (r *Repository) SaveSnapshot(sub model.PaperSubmission) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	filename := filepath.Join(r.snapshotDir, fmt.Sprintf("submission_%s_%s.json", sub.PaperID, sub.StudentID))
	data, err := json.MarshalIndent(sub, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func (r *Repository) SaveExportTask(task model.ExportTask) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.exportTasks[task.ID] = task
}

func (r *Repository) GetExportTask(id string) (model.ExportTask, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.exportTasks[id]
	return t, ok
}

func (r *Repository) UpdateExportTask(task model.ExportTask) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.exportTasks[task.ID] = task
}

func (r *Repository) AddSubmission(sub model.PaperSubmission) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.submissions[sub.ID] = sub
}

func (r *Repository) LockSubmission(subID, teacher string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	sub, ok := r.submissions[subID]
	if !ok {
		return false
	}
	if sub.LockedBy != "" && sub.LockedBy != teacher {
		return false
	}
	now := time.Now()
	sub.LockedBy = teacher
	sub.LockedAt = &now
	r.submissions[subID] = sub
	return true
}

func (r *Repository) UnlockSubmission(subID, teacher string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	sub, ok := r.submissions[subID]
	if !ok {
		return false
	}
	if sub.LockedBy != teacher {
		return false
	}
	sub.LockedBy = ""
	sub.LockedAt = nil
	r.submissions[subID] = sub
	return true
}
