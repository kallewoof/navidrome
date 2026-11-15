package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

type tagOverlayRepository struct {
	sqlRepository
}

type dbTagOverlay struct {
	MediaFileID string    `db:"media_file_id"`
	Tags        string    `db:"tags"`
	UpdatedAt   time.Time `db:"updated_at"`
	UpdatedBy   string    `db:"updated_by"`
}

func NewTagOverlayRepository(ctx context.Context, db dbx.Builder) model.TagOverlayRepository {
	r := &tagOverlayRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "tag_overlay"
	return r
}

func (o dbTagOverlay) toModel() (model.TagOverlay, error) {
	var tags map[string][]string
	if err := json.Unmarshal([]byte(o.Tags), &tags); err != nil {
		return model.TagOverlay{}, err
	}
	return model.TagOverlay{MediaFileID: o.MediaFileID, Tags: tags, UpdatedAt: o.UpdatedAt, UpdatedBy: o.UpdatedBy}, nil
}

func (r *tagOverlayRepository) Get(mediaFileID string) (*model.TagOverlay, error) {
	var row dbTagOverlay
	if err := r.queryOne(Select("*").From(r.tableName).Where(Eq{"media_file_id": mediaFileID}), &row); err != nil {
		return nil, err
	}
	o, err := row.toModel()
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *tagOverlayRepository) GetAll() ([]model.TagOverlay, error) {
	return r.list(Select("*").From(r.tableName).OrderBy("updated_at"))
}

func (r *tagOverlayRepository) GetMany(mediaFileIDs ...string) (map[string]model.TagOverlay, error) {
	out := map[string]model.TagOverlay{}
	if len(mediaFileIDs) == 0 {
		return out, nil
	}
	list, err := r.list(Select("*").From(r.tableName).Where(Eq{"media_file_id": mediaFileIDs}))
	if err != nil {
		return nil, err
	}
	for _, o := range list {
		out[o.MediaFileID] = o
	}
	return out, nil
}

func (r *tagOverlayRepository) list(sel SelectBuilder) ([]model.TagOverlay, error) {
	var rows []dbTagOverlay
	if err := r.queryAll(sel, &rows); err != nil && !errors.Is(err, model.ErrNotFound) {
		return nil, err
	}
	out := make([]model.TagOverlay, 0, len(rows))
	for _, row := range rows {
		o, err := row.toModel()
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, nil
}

func (r *tagOverlayRepository) Put(o *model.TagOverlay) error {
	tags, err := json.Marshal(o.Tags)
	if err != nil {
		return err
	}
	o.UpdatedAt = time.Now()
	upsert := Insert(r.tableName).Columns("media_file_id", "tags", "updated_at", "updated_by").
		Values(o.MediaFileID, string(tags), o.UpdatedAt, o.UpdatedBy).
		Suffix("on conflict (media_file_id) do update set tags = excluded.tags, updated_at = excluded.updated_at, updated_by = excluded.updated_by")
	_, err = r.executeSQL(upsert)
	return err
}

func (r *tagOverlayRepository) Delete(mediaFileID string) error {
	_, err := r.executeSQL(Delete(r.tableName).Where(Eq{"media_file_id": mediaFileID}))
	return err
}

func (r *tagOverlayRepository) MarkForRescan(mediaFileID string) error {
	folder := Update("folder").Set("hash", "").
		Where(Expr("id = (select folder_id from media_file where id = ?)", mediaFileID))
	if _, err := r.executeSQL(folder); err != nil {
		return err
	}
	_, err := r.executeSQL(Update("media_file").Set("updated_at", time.Time{}).Where(Eq{"id": mediaFileID}))
	return err
}
