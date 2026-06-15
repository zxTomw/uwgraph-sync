package neo4jstore

import (
	"context"
	"errors"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"uwgraph/internal/knowledge"
)

const (
	KnowledgeFullTextIndex = "knowledge_document_text"
	KnowledgeVectorIndex   = "knowledge_document_embedding"
)

func (s *Store) EnsureVectorIndex(ctx context.Context, dimensions int) error {
	if dimensions <= 0 {
		return errors.New("embedding dimensions must be greater than zero")
	}
	query := fmt.Sprintf(`
CREATE VECTOR INDEX %s IF NOT EXISTS
FOR (d:KnowledgeDocument) ON (d.embedding)
OPTIONS {indexConfig: {
  `+"`vector.dimensions`"+`: %d,
  `+"`vector.similarity_function`"+`: 'cosine'
}}`, KnowledgeVectorIndex, dimensions)
	if err := s.write(ctx, query, nil); err != nil {
		return fmt.Errorf("ensure vector index: %w", err)
	}
	return nil
}

func (s *Store) RebuildVectorIndex(ctx context.Context, dimensions int) error {
	if dimensions <= 0 {
		return errors.New("embedding dimensions must be greater than zero")
	}
	if err := s.write(ctx, "DROP INDEX "+KnowledgeVectorIndex+" IF EXISTS", nil); err != nil {
		return fmt.Errorf("drop vector index: %w", err)
	}
	if err := s.write(ctx, `
MATCH (d:KnowledgeDocument)
REMOVE d.embedding, d.embeddingHash, d.embeddingModel, d.embeddedAt`, nil); err != nil {
		return fmt.Errorf("clear embeddings: %w", err)
	}
	return s.EnsureVectorIndex(ctx, dimensions)
}

func (s *Store) PendingDocuments(ctx context.Context, model string, limit int) ([]knowledge.PendingDocument, error) {
	if limit <= 0 {
		return nil, errors.New("pending document limit must be greater than zero")
	}
	result, err := s.read(ctx, `
MATCH (d:KnowledgeDocument)
WHERE d.contentHash IS NOT NULL
  AND (
    d.embeddingHash IS NULL
    OR d.embeddingHash <> d.contentHash
    OR d.embeddingModel IS NULL
    OR d.embeddingModel <> $model
  )
RETURN d.documentKey AS documentKey,
       d.text AS text,
       d.contentHash AS contentHash
ORDER BY d.documentKey
LIMIT $limit`, map[string]any{
		"model": model,
		"limit": limit,
	})
	if err != nil {
		return nil, fmt.Errorf("query pending documents: %w", err)
	}

	documents := make([]knowledge.PendingDocument, 0, len(result.Records))
	for _, record := range result.Records {
		documentKey, err := requiredString(record, "documentKey")
		if err != nil {
			return nil, err
		}
		text, err := requiredString(record, "text")
		if err != nil {
			return nil, err
		}
		contentHash, err := requiredString(record, "contentHash")
		if err != nil {
			return nil, err
		}
		documents = append(documents, knowledge.PendingDocument{
			DocumentKey: documentKey,
			Text:        text,
			ContentHash: contentHash,
		})
	}
	return documents, nil
}

func (s *Store) ApplyEmbeddings(ctx context.Context, updates []knowledge.EmbeddingUpdate) (int, error) {
	rows := make([]map[string]any, 0, len(updates))
	for _, update := range updates {
		rows = append(rows, map[string]any{
			"documentKey": update.DocumentKey,
			"contentHash": update.ContentHash,
			"model":       update.Model,
			"embedding":   update.Embedding,
			"embeddedAt":  update.EmbeddedAt.UTC().Format(timeFormat),
		})
	}
	result, err := s.readWrite(ctx, `
UNWIND $rows AS row
MATCH (d:KnowledgeDocument {documentKey: row.documentKey})
WHERE d.contentHash = row.contentHash
SET d.embedding = row.embedding,
    d.embeddingHash = row.contentHash,
    d.embeddingModel = row.model,
    d.embeddedAt = row.embeddedAt
RETURN count(d) AS updated`, map[string]any{"rows": rows})
	if err != nil {
		return 0, fmt.Errorf("apply embeddings: %w", err)
	}
	if len(result.Records) == 0 {
		return 0, nil
	}
	updated, err := requiredInt64(result.Records[0], "updated")
	if err != nil {
		return 0, err
	}
	return int(updated), nil
}

func (s *Store) KnowledgeIndexesReady(ctx context.Context) error {
	result, err := s.read(ctx, `
SHOW INDEXES
YIELD name, state
WHERE name IN $names
RETURN name, state`, map[string]any{
		"names": []string{KnowledgeFullTextIndex, KnowledgeVectorIndex},
	})
	if err != nil {
		return fmt.Errorf("show knowledge indexes: %w", err)
	}
	states := make(map[string]string, len(result.Records))
	for _, record := range result.Records {
		name, err := requiredString(record, "name")
		if err != nil {
			return err
		}
		state, err := requiredString(record, "state")
		if err != nil {
			return err
		}
		states[name] = state
	}
	for _, name := range []string{KnowledgeFullTextIndex, KnowledgeVectorIndex} {
		if states[name] != "ONLINE" {
			return fmt.Errorf("knowledge index %s is not online", name)
		}
	}
	return nil
}

func (s *Store) FullTextCandidates(
	ctx context.Context,
	query string,
	kinds []string,
	termCode string,
	limit int,
) ([]knowledge.Candidate, error) {
	result, err := s.read(ctx, `
CALL db.index.fulltext.queryNodes($indexName, $query, {limit: $limit})
YIELD node, score
WHERE (size($kinds) = 0 OR node.kind IN $kinds)
  AND (
    $termCode = ''
    OR node.kind <> 'course'
    OR EXISTS {
      MATCH (node)-[:DESCRIBES]->(:Course)<-[:INSTANCE_OF]-(:CourseOffering)-[:IN_TERM]->(:Term {termCode: $termCode})
    }
  )
MATCH (node)-[:DESCRIBES]->(entity)
RETURN node.documentKey AS documentKey,
       node.entityUri AS entityUri,
       node.kind AS kind,
       node.sourceEntityKey AS entityKey,
       node.title AS title,
       node.text AS matchedText,
       node.sourceEndpoint AS sourceEndpoint,
       node.syncedAt AS syncedAt,
       properties(entity) AS facts,
       score
ORDER BY score DESC`, map[string]any{
		"indexName": KnowledgeFullTextIndex,
		"query":     query,
		"kinds":     kinds,
		"termCode":  termCode,
		"limit":     limit,
	})
	if err != nil {
		return nil, fmt.Errorf("full-text knowledge search: %w", err)
	}
	return candidatesFromResult(result)
}

func (s *Store) VectorCandidates(
	ctx context.Context,
	embedding []float32,
	kinds []string,
	termCode string,
	limit int,
) ([]knowledge.Candidate, error) {
	result, err := s.read(ctx, `
CALL db.index.vector.queryNodes($indexName, $limit, $embedding)
YIELD node, score
WHERE node.embeddingHash = node.contentHash
  AND (size($kinds) = 0 OR node.kind IN $kinds)
  AND (
    $termCode = ''
    OR node.kind <> 'course'
    OR EXISTS {
      MATCH (node)-[:DESCRIBES]->(:Course)<-[:INSTANCE_OF]-(:CourseOffering)-[:IN_TERM]->(:Term {termCode: $termCode})
    }
  )
MATCH (node)-[:DESCRIBES]->(entity)
RETURN node.documentKey AS documentKey,
       node.entityUri AS entityUri,
       node.kind AS kind,
       node.sourceEntityKey AS entityKey,
       node.title AS title,
       node.text AS matchedText,
       node.sourceEndpoint AS sourceEndpoint,
       node.syncedAt AS syncedAt,
       properties(entity) AS facts,
       score
ORDER BY score DESC`, map[string]any{
		"indexName": KnowledgeVectorIndex,
		"embedding": embedding,
		"kinds":     kinds,
		"termCode":  termCode,
		"limit":     limit,
	})
	if err != nil {
		return nil, fmt.Errorf("vector knowledge search: %w", err)
	}
	return candidatesFromResult(result)
}

func candidatesFromResult(result *neo4j.EagerResult) ([]knowledge.Candidate, error) {
	candidates := make([]knowledge.Candidate, 0, len(result.Records))
	for _, record := range result.Records {
		entityURI, err := requiredString(record, "entityUri")
		if err != nil {
			return nil, err
		}
		kind, err := requiredString(record, "kind")
		if err != nil {
			return nil, err
		}
		entityKey, err := requiredString(record, "entityKey")
		if err != nil {
			return nil, err
		}
		title, err := requiredString(record, "title")
		if err != nil {
			return nil, err
		}
		matchedText, err := requiredString(record, "matchedText")
		if err != nil {
			return nil, err
		}
		sourceEndpoint, err := requiredString(record, "sourceEndpoint")
		if err != nil {
			return nil, err
		}
		syncedAt, err := requiredString(record, "syncedAt")
		if err != nil {
			return nil, err
		}
		facts, err := requiredMap(record, "facts")
		if err != nil {
			return nil, err
		}
		score, err := requiredFloat64(record, "score")
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, knowledge.Candidate{
			Evidence: knowledge.Evidence{
				EntityURI:       entityURI,
				Kind:            kind,
				EntityKey:       entityKey,
				Title:           title,
				MatchedText:     matchedText,
				SourceEndpoint:  sourceEndpoint,
				SyncedAt:        syncedAt,
				Facts:           facts,
				RequirementsRaw: kind == knowledge.KindCourse && stringValue(facts["requirementsDescription"]) != "",
			},
			Score: score,
		})
	}
	return candidates, nil
}

func (s *Store) read(ctx context.Context, query string, params map[string]any) (*neo4j.EagerResult, error) {
	return neo4j.ExecuteQuery(
		ctx,
		s.driver,
		query,
		params,
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(s.database),
		neo4j.ExecuteQueryWithReadersRouting(),
	)
}

func (s *Store) readWrite(ctx context.Context, query string, params map[string]any) (*neo4j.EagerResult, error) {
	return neo4j.ExecuteQuery(
		ctx,
		s.driver,
		query,
		params,
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(s.database),
		neo4j.ExecuteQueryWithWritersRouting(),
	)
}

func requiredString(record *neo4j.Record, key string) (string, error) {
	value, isNil, err := neo4j.GetRecordValue[string](record, key)
	if err != nil {
		return "", err
	}
	if isNil {
		return "", fmt.Errorf("record value %s is null", key)
	}
	return value, nil
}

func requiredInt64(record *neo4j.Record, key string) (int64, error) {
	value, isNil, err := neo4j.GetRecordValue[int64](record, key)
	if err != nil {
		return 0, err
	}
	if isNil {
		return 0, fmt.Errorf("record value %s is null", key)
	}
	return value, nil
}

func requiredFloat64(record *neo4j.Record, key string) (float64, error) {
	value, isNil, err := neo4j.GetRecordValue[float64](record, key)
	if err != nil {
		return 0, err
	}
	if isNil {
		return 0, fmt.Errorf("record value %s is null", key)
	}
	return value, nil
}

func requiredMap(record *neo4j.Record, key string) (map[string]any, error) {
	value, isNil, err := neo4j.GetRecordValue[map[string]any](record, key)
	if err != nil {
		return nil, err
	}
	if isNil {
		return nil, fmt.Errorf("record value %s is null", key)
	}
	return value, nil
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

const timeFormat = "2006-01-02T15:04:05.999999999Z07:00"
