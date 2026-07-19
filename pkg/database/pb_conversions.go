// file: pkg/database/pb_conversions.go
// version: 4.0.0
// guid: c9cf2d1a-c284-46f4-90d1-70925cbe8b27

package database

import (
	"time"

	database "github.com/falkcorp/gcommon/v2/pkg/databasepb/v2"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// ToProto converts a SubtitleRecord to its gcommon Row representation.
func (r *SubtitleRecord) ToProto() *database.Row {
	if r == nil {
		return nil
	}

	row := &database.Row{}

	// Convert fields to Any values for the generic Row
	values := make([]*anypb.Any, 0)

	// Add ID
	if idAny, err := anypb.New(wrapperspb.String(r.ID)); err == nil {
		values = append(values, idAny)
	}

	// Add File
	if fileAny, err := anypb.New(wrapperspb.String(r.File)); err == nil {
		values = append(values, fileAny)
	}

	// Add VideoFile
	if videoAny, err := anypb.New(wrapperspb.String(r.VideoFile)); err == nil {
		values = append(values, videoAny)
	}

	// Add Language
	if langAny, err := anypb.New(wrapperspb.String(r.Language)); err == nil {
		values = append(values, langAny)
	}

	// Add Release
	if releaseAny, err := anypb.New(wrapperspb.String(r.Release)); err == nil {
		values = append(values, releaseAny)
	}

	// Add Service
	if serviceAny, err := anypb.New(wrapperspb.String(r.Service)); err == nil {
		values = append(values, serviceAny)
	}

	// Add Embedded
	if embeddedAny, err := anypb.New(wrapperspb.Bool(r.Embedded)); err == nil {
		values = append(values, embeddedAny)
	}

	// Add SourceURL
	if sourceAny, err := anypb.New(wrapperspb.String(r.SourceURL)); err == nil {
		values = append(values, sourceAny)
	}

	// Add ProviderMetadata
	if metadataAny, err := anypb.New(wrapperspb.String(r.ProviderMetadata)); err == nil {
		values = append(values, metadataAny)
	}

	// Add ConfidenceScore (nullable)
	if r.ConfidenceScore != nil {
		if scoreAny, err := anypb.New(wrapperspb.Double(*r.ConfidenceScore)); err == nil {
			values = append(values, scoreAny)
		}
	} else {
		// Use empty Any for null values
		values = append(values, &anypb.Any{})
	}

	// Add ParentID (nullable)
	if r.ParentID != nil {
		if parentAny, err := anypb.New(wrapperspb.String(*r.ParentID)); err == nil {
			values = append(values, parentAny)
		}
	} else {
		// Use empty Any for null values
		values = append(values, &anypb.Any{})
	}

	// Add ModificationType
	if modAny, err := anypb.New(wrapperspb.String(r.ModificationType)); err == nil {
		values = append(values, modAny)
	}

	// Add CreatedAt
	if timeAny, err := anypb.New(timestamppb.New(r.CreatedAt.UTC())); err == nil {
		values = append(values, timeAny)
	}

	row.SetValues(values)
	return row
}

// SubtitleRecordFromProto converts a gcommon Row to an internal SubtitleRecord struct.
func SubtitleRecordFromProto(row *database.Row) *SubtitleRecord {
	if row == nil {
		return nil
	}

	values := row.GetValues()
	if len(values) < 12 { // Need at least 12 values for complete record
		return nil
	}

	rec := &SubtitleRecord{}

	// Extract ID
	if len(values) > 0 && values[0] != nil {
		msg := &wrapperspb.StringValue{}
		if values[0].UnmarshalTo(msg) == nil {
			rec.ID = msg.Value
		}
	}

	// Extract File
	if len(values) > 1 && values[1] != nil {
		msg := &wrapperspb.StringValue{}
		if values[1].UnmarshalTo(msg) == nil {
			rec.File = msg.Value
		}
	}

	// Extract VideoFile
	if len(values) > 2 && values[2] != nil {
		msg := &wrapperspb.StringValue{}
		if values[2].UnmarshalTo(msg) == nil {
			rec.VideoFile = msg.Value
		}
	}

	// Extract Language
	if len(values) > 3 && values[3] != nil {
		msg := &wrapperspb.StringValue{}
		if values[3].UnmarshalTo(msg) == nil {
			rec.Language = msg.Value
		}
	}

	// Extract Release
	if len(values) > 4 && values[4] != nil {
		msg := &wrapperspb.StringValue{}
		if values[4].UnmarshalTo(msg) == nil {
			rec.Release = msg.Value
		}
	}

	// Extract Service
	if len(values) > 5 && values[5] != nil {
		msg := &wrapperspb.StringValue{}
		if values[5].UnmarshalTo(msg) == nil {
			rec.Service = msg.Value
		}
	}

	// Extract Embedded
	if len(values) > 6 && values[6] != nil {
		msg := &wrapperspb.BoolValue{}
		if values[6].UnmarshalTo(msg) == nil {
			rec.Embedded = msg.Value
		}
	}

	// Extract SourceURL
	if len(values) > 7 && values[7] != nil {
		msg := &wrapperspb.StringValue{}
		if values[7].UnmarshalTo(msg) == nil {
			rec.SourceURL = msg.Value
		}
	}

	// Extract ProviderMetadata
	if len(values) > 8 && values[8] != nil {
		msg := &wrapperspb.StringValue{}
		if values[8].UnmarshalTo(msg) == nil {
			rec.ProviderMetadata = msg.Value
		}
	}

	// Extract ConfidenceScore (nullable)
	if len(values) > 9 && values[9] != nil && len(values[9].Value) > 0 {
		msg := &wrapperspb.DoubleValue{}
		if values[9].UnmarshalTo(msg) == nil {
			rec.ConfidenceScore = &msg.Value
		}
	}

	// Extract ParentID (nullable)
	if len(values) > 10 && values[10] != nil && len(values[10].Value) > 0 {
		msg := &wrapperspb.StringValue{}
		if values[10].UnmarshalTo(msg) == nil {
			rec.ParentID = &msg.Value
		}
	}

	// Extract ModificationType
	if len(values) > 11 && values[11] != nil {
		msg := &wrapperspb.StringValue{}
		if values[11].UnmarshalTo(msg) == nil {
			rec.ModificationType = msg.Value
		}
	}

	// Extract CreatedAt
	if len(values) > 12 && values[12] != nil {
		ts := &timestamppb.Timestamp{}
		if values[12].UnmarshalTo(ts) == nil {
			rec.CreatedAt = ts.AsTime()
		} else {
			rec.CreatedAt = time.Now()
		}
	} else {
		rec.CreatedAt = time.Now()
	}

	return rec
}

// ToProto converts a DownloadRecord to its gcommon Row representation.
func (r *DownloadRecord) ToProto() *database.Row {
	if r == nil {
		return nil
	}

	row := &database.Row{}
	values := make([]*anypb.Any, 0)

	// Add ID
	if idAny, err := anypb.New(wrapperspb.String(r.ID)); err == nil {
		values = append(values, idAny)
	}

	// Add File
	if fileAny, err := anypb.New(wrapperspb.String(r.File)); err == nil {
		values = append(values, fileAny)
	}

	// Add VideoFile
	if videoAny, err := anypb.New(wrapperspb.String(r.VideoFile)); err == nil {
		values = append(values, videoAny)
	}

	// Add Provider
	if providerAny, err := anypb.New(wrapperspb.String(r.Provider)); err == nil {
		values = append(values, providerAny)
	}

	// Add Language
	if langAny, err := anypb.New(wrapperspb.String(r.Language)); err == nil {
		values = append(values, langAny)
	}

	// Add SearchQuery
	if queryAny, err := anypb.New(wrapperspb.String(r.SearchQuery)); err == nil {
		values = append(values, queryAny)
	}

	// Add MatchScore (nullable)
	if r.MatchScore != nil {
		if scoreAny, err := anypb.New(wrapperspb.Double(*r.MatchScore)); err == nil {
			values = append(values, scoreAny)
		}
	} else {
		values = append(values, &anypb.Any{})
	}

	// Add DownloadAttempts
	if attemptsAny, err := anypb.New(wrapperspb.Int32(int32(r.DownloadAttempts))); err == nil {
		values = append(values, attemptsAny)
	}

	// Add ErrorMessage
	if errorAny, err := anypb.New(wrapperspb.String(r.ErrorMessage)); err == nil {
		values = append(values, errorAny)
	}

	// Add ResponseTimeMs (nullable)
	if r.ResponseTimeMs != nil {
		if timeAny, err := anypb.New(wrapperspb.Int32(int32(*r.ResponseTimeMs))); err == nil {
			values = append(values, timeAny)
		}
	} else {
		values = append(values, &anypb.Any{})
	}

	// Add CreatedAt
	if createdAny, err := anypb.New(timestamppb.New(r.CreatedAt.UTC())); err == nil {
		values = append(values, createdAny)
	}

	row.SetValues(values)
	return row
}

// DownloadRecordFromProto converts a gcommon Row to an internal DownloadRecord struct.
func DownloadRecordFromProto(row *database.Row) *DownloadRecord {
	if row == nil {
		return nil
	}

	values := row.GetValues()
	if len(values) < 10 { // Need at least 10 values for complete record
		return nil
	}

	rec := &DownloadRecord{}

	// Extract ID
	if len(values) > 0 && values[0] != nil {
		msg := &wrapperspb.StringValue{}
		if values[0].UnmarshalTo(msg) == nil {
			rec.ID = msg.Value
		}
	}

	// Extract File
	if len(values) > 1 && values[1] != nil {
		msg := &wrapperspb.StringValue{}
		if values[1].UnmarshalTo(msg) == nil {
			rec.File = msg.Value
		}
	}

	// Extract VideoFile
	if len(values) > 2 && values[2] != nil {
		msg := &wrapperspb.StringValue{}
		if values[2].UnmarshalTo(msg) == nil {
			rec.VideoFile = msg.Value
		}
	}

	// Extract Provider
	if len(values) > 3 && values[3] != nil {
		msg := &wrapperspb.StringValue{}
		if values[3].UnmarshalTo(msg) == nil {
			rec.Provider = msg.Value
		}
	}

	// Extract Language
	if len(values) > 4 && values[4] != nil {
		msg := &wrapperspb.StringValue{}
		if values[4].UnmarshalTo(msg) == nil {
			rec.Language = msg.Value
		}
	}

	// Extract SearchQuery
	if len(values) > 5 && values[5] != nil {
		msg := &wrapperspb.StringValue{}
		if values[5].UnmarshalTo(msg) == nil {
			rec.SearchQuery = msg.Value
		}
	}

	// Extract MatchScore (nullable)
	if len(values) > 6 && values[6] != nil && len(values[6].Value) > 0 {
		msg := &wrapperspb.DoubleValue{}
		if values[6].UnmarshalTo(msg) == nil {
			rec.MatchScore = &msg.Value
		}
	}

	// Extract DownloadAttempts
	if len(values) > 7 && values[7] != nil {
		msg := &wrapperspb.Int32Value{}
		if values[7].UnmarshalTo(msg) == nil {
			rec.DownloadAttempts = int(msg.Value)
		}
	}

	// Extract ErrorMessage
	if len(values) > 8 && values[8] != nil {
		msg := &wrapperspb.StringValue{}
		if values[8].UnmarshalTo(msg) == nil {
			rec.ErrorMessage = msg.Value
		}
	}

	// Extract ResponseTimeMs (nullable)
	if len(values) > 9 && values[9] != nil && len(values[9].Value) > 0 {
		msg := &wrapperspb.Int32Value{}
		if values[9].UnmarshalTo(msg) == nil {
			intVal := int(msg.Value)
			rec.ResponseTimeMs = &intVal
		}
	}

	// Extract CreatedAt
	if len(values) > 10 && values[10] != nil {
		ts := &timestamppb.Timestamp{}
		if values[10].UnmarshalTo(ts) == nil {
			rec.CreatedAt = ts.AsTime()
		} else {
			rec.CreatedAt = time.Now()
		}
	} else {
		rec.CreatedAt = time.Now()
	}

	return rec
}
