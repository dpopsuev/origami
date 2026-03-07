package rca

import (
	"context"
	"fmt"

	framework "github.com/dpopsuev/origami"
	"github.com/dpopsuev/origami/schematics/rca/rcatype"
	"github.com/dpopsuev/origami/schematics/rca/store"
	"github.com/dpopsuev/origami/schematics/toolkit"
)

// Context keys used by inject hooks to store assembled template data.
const (
	KeyParamsEnvelope = "params.envelope"
	KeyParamsFailure  = "params.failure"
	KeyParamsHistory  = "params.history"
	KeyParamsDigest   = "params.recall_digest"
	KeyParamsSources  = "params.sources"
	KeyParamsPrior    = "params.prior"
	KeyParamsTaxonomy = "params.taxonomy"
	KeyParamsCode     = "params.code"
)

const maxCodeTokenBudget = 32000

// InjectHookOpts configures the inject hook registry.
type InjectHookOpts struct {
	Store           store.Store
	CaseData        *store.Case
	Envelope        *rcatype.Envelope
	Catalog         toolkit.SourceCatalog
	CaseDir         string
	KnowledgeReader toolkit.SourceReader
}

// InjectHooks creates a HookRegistry with the inject.* before-hooks
// that populate walker.Context with per-concern template data.
// Each hook uses WalkerStateFromContext to write into walker.Context.
func InjectHooks(st store.Store, caseData *store.Case, env *rcatype.Envelope, catalog toolkit.SourceCatalog, caseDir string) framework.HookRegistry {
	return InjectHooksWithOpts(InjectHookOpts{
		Store:    st,
		CaseData: caseData,
		Envelope: env,
		Catalog:  catalog,
		CaseDir:  caseDir,
	})
}

// InjectHooksWithOpts creates inject hooks using the full options struct,
// including optional KnowledgeReader for code injection hooks.
func InjectHooksWithOpts(opts InjectHookOpts) framework.HookRegistry {
	reg := framework.HookRegistry{}

	reg.Register(newInjectEnvelopeHook(opts.Envelope))
	reg.Register(newInjectFailureHook(opts.CaseData))
	reg.Register(newInjectHistoryHook(opts.Store, opts.CaseData))
	reg.Register(newInjectRecallDigestHook(opts.Store))
	reg.Register(newInjectSourcesHook(opts.Envelope, opts.Catalog))
	reg.Register(newInjectPriorHook(opts.CaseDir))
	reg.Register(newInjectTaxonomyHook())

	if opts.KnowledgeReader != nil && opts.Catalog != nil {
		reg.Register(newInjectCodeTreeHook(opts.KnowledgeReader, opts.Catalog))
		reg.Register(newInjectCodeSearchHook(opts.KnowledgeReader, opts.Catalog))
		reg.Register(newInjectCodeReadHook(opts.KnowledgeReader))
	}

	return reg
}

func newInjectEnvelopeHook(env *rcatype.Envelope) framework.Hook {
	return toolkit.NewContextInjector("inject.envelope", func(walkerCtx map[string]any) {
		injectEnvelopeData(env, walkerCtx)
	})
}

func newInjectFailureHook(caseData *store.Case) framework.Hook {
	return toolkit.NewContextInjector("inject.failure", func(walkerCtx map[string]any) {
		injectFailureData(caseData, walkerCtx)
	})
}

func newInjectHistoryHook(st store.Store, caseData *store.Case) framework.Hook {
	return toolkit.NewContextInjector("inject.history", func(walkerCtx map[string]any) {
		injectHistoryData(st, caseData, walkerCtx)
	})
}

func newInjectRecallDigestHook(st store.Store) framework.Hook {
	return toolkit.NewContextInjector("inject.recall-digest", func(walkerCtx map[string]any) {
		injectRecallDigestData(st, walkerCtx)
	})
}

func newInjectSourcesHook(env *rcatype.Envelope, catalog toolkit.SourceCatalog) framework.Hook {
	return toolkit.NewContextInjector("inject.sources", func(walkerCtx map[string]any) {
		injectSourcesData(env, catalog, walkerCtx)
	})
}

func newInjectPriorHook(caseDir string) framework.Hook {
	return toolkit.NewContextInjector("inject.prior", func(walkerCtx map[string]any) {
		injectPriorData(caseDir, walkerCtx)
	})
}

func newInjectTaxonomyHook() framework.Hook {
	return toolkit.NewContextInjector("inject.taxonomy", func(walkerCtx map[string]any) {
		injectTaxonomyData(walkerCtx)
	})
}

// ParamsFromContext assembles a TemplateParams from walker context.
// Before-hooks inject their data into walker.Context with keys like
// "params.envelope", "params.failure", etc. This function collects
// them into the TemplateParams structure that templates expect.
func ParamsFromContext(walkerCtx map[string]any) *TemplateParams {
	params := &TemplateParams{}

	if v, ok := walkerCtx[KeyParamsEnvelope].(*EnvelopeParams); ok {
		params.Envelope = v
		params.SourceID = v.RunID
	}

	if v, ok := walkerCtx[KeyParamsFailure].(*FailureParams); ok {
		params.Failure = v
	}

	if v, ok := walkerCtx[KeyParamsSources].(*SourceParams); ok {
		params.Sources = v
	}

	if v, ok := walkerCtx[KeyParamsHistory].(*HistoryParams); ok {
		params.History = v
	}

	if v, ok := walkerCtx[KeyParamsDigest].([]RecallDigestEntry); ok {
		params.RecallDigest = v
	}

	if v, ok := walkerCtx[KeyParamsPrior].(*PriorParams); ok {
		params.Prior = v
	}

	if v, ok := walkerCtx[KeyParamsTaxonomy].(*TaxonomyParams); ok {
		params.Taxonomy = v
	}

	if v, ok := walkerCtx[KeyParamsCode].(*CodeParams); ok {
		params.Code = v
	}

	if cd, ok := walkerCtx[KeyCaseData].(*store.Case); ok {
		params.CaseID = cd.ID
	}

	if _, ok := walkerCtx[KeyParamsEnvelope].(*EnvelopeParams); ok {
		if env, ok := walkerCtx[KeyEnvelope].(*rcatype.Envelope); ok {
			for _, f := range env.FailureList {
				params.Siblings = append(params.Siblings, SiblingParams{
					ID: f.ID, Name: f.Name, Status: f.Status,
				})
			}
		}
	}

	if params.Timestamps == nil {
		params.Timestamps = &TimestampParams{
			ClockPlaneNote: "Note: Timestamps may originate from different clock planes (executor, test node, SUT). Cross-plane time comparisons may be unreliable.",
		}
	}

	return params
}

// Concrete implementations that actually inject data into walker context.

func injectEnvelopeData(env *rcatype.Envelope, walkerCtx map[string]any) {
	if env == nil {
		return
	}
	walkerCtx[KeyParamsEnvelope] = &EnvelopeParams{
		Name:  env.Name,
		RunID: env.RunID,
	}
}

func injectFailureData(caseData *store.Case, walkerCtx map[string]any) {
	if caseData == nil {
		return
	}
	walkerCtx[KeyParamsFailure] = &FailureParams{
		TestName:     caseData.Name,
		ErrorMessage: caseData.ErrorMessage,
		LogSnippet:   caseData.LogSnippet,
		LogTruncated: caseData.LogTruncated,
		Status:       caseData.Status,
	}
}

func injectHistoryData(st store.Store, caseData *store.Case, walkerCtx map[string]any) {
	if st == nil || caseData == nil {
		return
	}
	if caseData.SymptomID != 0 {
		walkerCtx[KeyParamsHistory] = loadHistory(st, caseData.SymptomID)
	} else {
		walkerCtx[KeyParamsHistory] = findRecallCandidates(st, caseData.Name)
	}
}

func injectRecallDigestData(st store.Store, walkerCtx map[string]any) {
	if st == nil {
		return
	}
	walkerCtx[KeyParamsDigest] = buildRecallDigest(st)
}

func injectSourcesData(env *rcatype.Envelope, catalog toolkit.SourceCatalog, walkerCtx map[string]any) {
	walkerCtx[KeyParamsSources] = buildSourceParams(env, catalog)
}

func injectPriorData(caseDir string, walkerCtx map[string]any) {
	if caseDir == "" {
		return
	}
	walkerCtx[KeyParamsPrior] = loadPriorArtifacts(caseDir)
}

func injectTaxonomyData(walkerCtx map[string]any) {
	walkerCtx[KeyParamsTaxonomy] = DefaultTaxonomy()
}

// Code injection hooks

func newInjectCodeTreeHook(reader toolkit.SourceReader, catalog toolkit.SourceCatalog) framework.Hook {
	return toolkit.NewContextInjectorErr("inject.code.tree", func(ctx context.Context, walkerCtx map[string]any) error {
		code := ensureCodeParams(walkerCtx)
		for _, src := range catalog.Sources() {
			if src.Kind != toolkit.SourceKindRepo {
				continue
			}
			if err := reader.Ensure(ctx, src); err != nil {
				continue
			}
			entries, err := reader.List(ctx, src, "", 3)
			if err != nil {
				continue
			}
			var treeEntries []TreeEntry
			for _, e := range entries {
				treeEntries = append(treeEntries, TreeEntry{Path: e.Path, IsDir: e.IsDir})
			}
			code.Trees = append(code.Trees, CodeTreeParams{
				Repo:    fmt.Sprintf("%s/%s", src.Org, src.Name),
				Branch:  src.Branch,
				Entries: treeEntries,
			})
		}
		return nil
	})
}

func newInjectCodeSearchHook(reader toolkit.SourceReader, catalog toolkit.SourceCatalog) framework.Hook {
	return toolkit.NewContextInjectorErr("inject.code.search", func(ctx context.Context, walkerCtx map[string]any) error {
		code := ensureCodeParams(walkerCtx)

		keywords := extractSearchKeywords(walkerCtx)
		if len(keywords) == 0 {
			return nil
		}

		for _, src := range catalog.Sources() {
			if src.Kind != toolkit.SourceKindRepo {
				continue
			}
			query := keywords[0]
			for _, kw := range keywords[1:] {
				query += " " + kw
			}
			results, err := reader.Search(ctx, src, query, 20)
			if err != nil {
				continue
			}
			repoName := fmt.Sprintf("%s/%s", src.Org, src.Name)
			for _, r := range results {
				code.SearchResults = append(code.SearchResults, CodeSearchResult{
					Repo:    repoName,
					File:    r.Path,
					Line:    r.Line,
					Snippet: r.Snippet,
				})
			}
		}
		return nil
	})
}

func newInjectCodeReadHook(reader toolkit.SourceReader) framework.Hook {
	return toolkit.NewContextInjectorErr("inject.code.read", func(ctx context.Context, walkerCtx map[string]any) error {
		code := ensureCodeParams(walkerCtx)

		seen := make(map[string]bool)
		budgetRemaining := maxCodeTokenBudget
		for _, sr := range code.SearchResults {
			fileKey := sr.Repo + ":" + sr.File
			if seen[fileKey] {
				continue
			}
			seen[fileKey] = true

			parts := splitRepoKey(sr.Repo)
			if parts == nil {
				continue
			}
		src := toolkit.Source{
			Org:  parts[0],
			Name: parts[1],
			Kind: toolkit.SourceKindRepo,
		}
			data, err := reader.Read(ctx, src, sr.File)
			if err != nil {
				continue
			}

			content := string(data)
			truncated := false
			if len(content) > budgetRemaining {
				content = content[:budgetRemaining]
				truncated = true
			}
			budgetRemaining -= len(content)

			code.Files = append(code.Files, CodeFileParams{
				Repo:      sr.Repo,
				Path:      sr.File,
				Content:   content,
				Truncated: truncated,
			})

			if budgetRemaining <= 0 {
				code.Truncated = true
				break
			}
		}
		return nil
	})
}

func ensureCodeParams(walkerCtx map[string]any) *CodeParams {
	if v, ok := walkerCtx[KeyParamsCode].(*CodeParams); ok {
		return v
	}
	code := &CodeParams{}
	walkerCtx[KeyParamsCode] = code
	return code
}

func extractSearchKeywords(walkerCtx map[string]any) []string {
	var keywords []string
	if fp, ok := walkerCtx[KeyParamsFailure].(*FailureParams); ok && fp != nil {
		if fp.TestName != "" {
			keywords = append(keywords, fp.TestName)
		}
	}
	if prior, ok := walkerCtx[KeyParamsPrior].(*PriorParams); ok && prior != nil {
		if prior.Triage != nil {
			if repos, ok := prior.Triage["candidate_repos"].([]any); ok {
				for _, r := range repos {
					if s, ok := r.(string); ok {
						keywords = append(keywords, s)
					}
				}
			}
		}
		if prior.Resolve != nil {
			if repos, ok := prior.Resolve["selected_repos"].([]any); ok {
				for _, r := range repos {
					if rm, ok := r.(map[string]any); ok {
						if name, ok := rm["name"].(string); ok {
							keywords = append(keywords, name)
						}
					}
				}
			}
		}
	}
	return keywords
}

func splitRepoKey(key string) []string {
	for i, c := range key {
		if c == '/' {
			return []string{key[:i], key[i+1:]}
		}
	}
	return nil
}
