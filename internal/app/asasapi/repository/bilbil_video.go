package repository

import (
	"fmt"
	"strings"
	"time"

	"github.com/A-SoulFan/asasfans-api/internal/app/asasapi/idl"
	"github.com/A-SoulFan/asasfans-api/internal/app/asasapi/util/query_parser"
	"gorm.io/gorm/clause"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const (
	bilbilVideoTableName    = "bilbil_asoul_video"
	bilbilVideoTagTableName = "bilbil_video_tag"
)

func NewBilbilVideo(tx *gorm.DB) idl.BilbilVideoRepository {
	return &BilbilVideoMysqlImpl{tx: tx}
}

type BilbilVideoMysqlImpl struct {
	tx *gorm.DB
}

func (impl *BilbilVideoMysqlImpl) FindAllByPubDate(from, to time.Time, page, size int64) (list []*idl.BilbilVideo, total int64, err error) {
	result := impl.tx.Table(bilbilVideoTableName).
		Where("pubdate >= ? AND pubdate <= ?", from.Unix(), to.Unix()).
		Offset(int((page - 1) * size)).Limit(int(size)).
		Order("pubdate DESC").
		Find(&list)

	if result == nil {
		return nil, 0, errors.Wrap(result.Error, fmt.Sprintf("select from %s error", bilbilVideoTableName))
	}

	result = impl.tx.Table(bilbilVideoTableName).
		Select("id").
		Where("pubdate >= ? AND pubdate <= ?", from.Second(), to).
		Count(&total)

	if result == nil {
		return nil, 0, errors.Wrap(result.Error, fmt.Sprintf("count from %s error", bilbilVideoTableName))
	}

	return list, total, nil
}

func (impl *BilbilVideoMysqlImpl) Search(queryItems []query_parser.QueryItem, order idl.BilbilVideoOrder, page, size int64) (list []*idl.BilbilVideo, total int64, err error) {
	renameMap := map[string]string{
		"tag": fmt.Sprintf("%s.tag", bilbilVideoTagTableName),
	}

	resp := builderQueryItems(impl.tx, queryItems, renameMap).Table(bilbilVideoTableName).
		Joins(fmt.Sprintf("JOIN %s ON %s.bvid = %s.bvid", bilbilVideoTagTableName, bilbilVideoTagTableName, bilbilVideoTableName)).
		Select(fmt.Sprintf("%s.*", bilbilVideoTableName)).
		Order(fmt.Sprintf("%s DESC", order)).
		Offset(int((page - 1) * size)).Limit(int(size)).
		Find(&list)

	if resp.Error != nil {
		return nil, 0, errors.Wrap(resp.Error, fmt.Sprintf("select from %s error", bilbilVideoTableName))
	}

	resp = builderQueryItems(impl.tx, queryItems, renameMap).Table(bilbilVideoTableName).
		Joins(fmt.Sprintf("JOIN %s ON %s.bvid = %s.bvid", bilbilVideoTagTableName, bilbilVideoTagTableName, bilbilVideoTableName)).
		Select(fmt.Sprintf("%s.id", bilbilVideoTableName)).
		Count(&total)

	if resp.Error != nil {
		return nil, 0, errors.Wrap(resp.Error, fmt.Sprintf("count from %s error", bilbilVideoTableName))
	}

	return list, total, nil
}

func builderQueryItems(tx *gorm.DB, queryItems []query_parser.QueryItem, rename map[string]string) *gorm.DB {
	for _, item := range queryItems {
		key := item.Key

		if rename != nil {
			if newKey, ok := rename[key]; ok {
				key = newKey
			}
		}

		switch item.Type {
		case query_parser.TypeAND:
			for _, value := range item.Values {
				tx = tx.Where(fmt.Sprintf("%s = ?", key), value)
			}
		case query_parser.TypeOR:
			tx = tx.Where(fmt.Sprintf("%s IN (?)", key), item.Values)
		case query_parser.TypeBetween:
			tx = tx.Where(fmt.Sprintf("%s BETWEEN ? AND ?", key), item.GetBetweenValues())
		}
	}

	return tx
}

func (impl *BilbilVideoMysqlImpl) Create(e *idl.BilbilVideo) error {
	return impl.tx.Transaction(func(_tx *gorm.DB) error {
		result := _tx.Table(bilbilVideoTableName).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "bvid"}},
			UpdateAll: true,
		}).Create(&e)

		if result.Error != nil {
			return errors.Wrap(result.Error, fmt.Sprintf("insert %s fail", bilbilVideoTableName))
		}

		tags := strings.Split(e.Tag, ",")
		for _, tag := range tags {
			result = _tx.Table(bilbilVideoTagTableName).Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "bvid"}},
				DoNothing: true,
			}).Create(struct {
				Bvid string
				Tag  string
			}{Bvid: e.Bvid,
				Tag: tag})

			if result.Error != nil {
				return errors.Wrap(result.Error, fmt.Sprintf("insert %s fail", bilbilVideoTagTableName))
			}
		}

		return nil
	})
}
