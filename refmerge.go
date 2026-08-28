package walle

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
)

// Merging sibling constraints into a $ref target.

// lowerBoundKeywords tighten as the value grows.
var lowerBoundKeywords = map[string]bool{
	MinLength: true,
	MinItems:  true,
	Minimum:   true,
}

// upperBoundKeywords tighten as the value shrinks.
var upperBoundKeywords = map[string]bool{
	MaxLength: true,
	MaxItems:  true,
	Maximum:   true,
}

// refInlineBudget tracks the remaining copy allowance for one inlining pass,
// in serialized bytes. The budget is the headroom left under the schema size
// limit, so it is always the budget -- not the size check -- that decides when
// inlining stops; that keeps the degradation local instead of letting the size
// check discard the whole schema. Once a merge is refused the budget stays
// closed: the alternative, measuring every later target before refusing it,
// can itself be driven into quadratic time.
type refInlineBudget struct {
	remaining int
	exhausted bool
}

func (b *refInlineBudget) tryConsume(cost int) bool {
	if b.exhausted {
		return false
	}
	if cost > b.remaining {
		b.exhausted = true
		return false
	}
	b.remaining -= cost
	return true
}

// serializedSize approximates the bytes a value adds to the marshaled schema.
func serializedSize(node any) int {
	encoded, err := json.Marshal(node)
	if err != nil {
		return 0
	}
	return len(encoded)
}

// inlineConflictingRefSiblings rewrites every node that holds both a $ref and
// its own constraints, replacing it with the merged schema. The input is not
// modified; when there is nothing to merge the original is returned as is.
//
// Inlining copies the definition at every use site, so a DAG of definitions
// expands exponentially. Once the copy budget -- the headroom under the schema
// size limit -- runs out, the siblings that could not be folded in are stripped
// in one pass and reported, so the caller can warn about exactly what was lost.
func inlineConflictingRefSiblings(root SchemaDict, maxSchemaSize int) (SchemaDict, []string) {
	if len(root) == 0 || !hasMergeableRefSibling(root, root) {
		return root, nil
	}

	copied, ok := deepCopyValue(root).(SchemaDict)
	if !ok {
		return root, nil
	}

	budget := &refInlineBudget{remaining: maxSchemaSize - serializedSize(copied)}
	mergeRefSiblings(copied, copied, budget)
	pruneOrphanDefs(copied)
	if !budget.exhausted {
		return copied, nil
	}

	return copied, stripRefSiblings(copied, copied, nil)
}

// stripRefSiblings removes the sibling constraints of every node that was
// still mergeable when the copy budget ran out. This is the same drop
// validation would apply one error at a time; doing it in one pass keeps
// Canonical from exhausting its retry budget on large schemas and returning
// {} instead of a loosened schema. Nodes whose siblings contradict the
// referenced schema are left alone: validation merges those, keeping whatever
// the two sides agree on, which loses less than a wholesale drop. The result
// lists every dropped keyword with its location.
func stripRefSiblings(node any, root SchemaDict, path []string) []string {
	var dropped []string
	switch n := node.(type) {
	case SchemaDict:
		if ref, ok := n[Ref].(string); ok {
			if siblings, ok := mergeableSiblings(n, root, ref); ok {
				keys := make([]string, 0, len(siblings))
				for key := range siblings {
					keys = append(keys, key)
				}
				sort.Strings(keys)

				where := "root"
				if len(path) > 0 {
					where = newSchemaPathFromParts(path).String()
				}
				for _, key := range keys {
					delete(n, key)
					dropped = append(dropped, fmt.Sprintf("%s at %s", key, where))
				}
			}
		}
		for key, value := range n {
			dropped = append(dropped, stripRefSiblings(value, root, append(path, key))...)
		}
	case SchemaList:
		for i, item := range n {
			dropped = append(dropped, stripRefSiblings(item, root, append(path, fmt.Sprintf("{%d}", i)))...)
		}
	}
	return dropped
}

// mergeableSiblings returns the constraint keywords a node carries beside its
// $ref when they could safely have been folded in, mirroring what validation
// would drop: annotations and the root's own $defs / $id stay, and so does a
// node whose siblings contradict the referenced schema.
func mergeableSiblings(node SchemaDict, root SchemaDict, ref string) (SchemaDict, bool) {
	target, ok := resolveDefsRef(root, ref)
	if !ok || len(target) == 0 {
		return nil, false
	}

	siblings := make(SchemaDict, len(node)-1)
	for key, value := range node {
		if key == Ref || CommonKeywords[key] || (TopLevelOnlyKeywords[key] && sameSchemaDict(node, root)) {
			continue
		}
		siblings[key] = value
	}
	if len(siblings) == 0 {
		return nil, false
	}
	if unsatisfiableOverlap(siblings, target) != "" {
		return nil, false
	}
	return siblings, true
}

// distributeAnyOfParentKeywords rewrites every node that constrains an instance
// both directly and through anyOf. The two apply together, but the enforcer's
// anyOf is a plain union with no room for a conjunct beside it, so the parent
// constraint is pushed into each branch instead:
//
//	{minLength: 5, anyOf: [A, B]}  ==  {anyOf: [A and {minLength: 5}, B and {minLength: 5}]}
//
// A branch that cannot agree with the parent is dropped: no instance reaches it.
// Dropping every branch leaves a node nothing can satisfy, which degrades to {}
// rather than to an empty anyOf that would accept everything.
//
// The input is not modified; when there is nothing to distribute the original is
// returned as is.
func distributeAnyOfParentKeywords(root SchemaDict) SchemaDict {
	if len(root) == 0 || !hasDistributableAnyOf(root, root) {
		return root
	}

	copied, ok := deepCopyValue(root).(SchemaDict)
	if !ok {
		return root
	}

	distributeAnyOf(copied, copied)
	return copied
}

// hasDistributableAnyOf reports whether the subtree holds a node worth rewriting,
// so the common case costs a single read-only walk.
func hasDistributableAnyOf(node any, root SchemaDict) bool {
	switch n := node.(type) {
	case SchemaDict:
		if _, _, ok := distributableAnyOf(n, root); ok {
			return true
		}
		for _, value := range n {
			if hasDistributableAnyOf(value, root) {
				return true
			}
		}
	case SchemaList:
		for _, item := range n {
			if hasDistributableAnyOf(item, root) {
				return true
			}
		}
	}
	return false
}

// distributeAnyOf walks the tree and rewrites every distributable node in place.
func distributeAnyOf(node any, root SchemaDict) {
	switch n := node.(type) {
	case SchemaDict:
		if branches, outer, ok := distributableAnyOf(n, root); ok {
			merged := make(SchemaList, 0, len(branches))
			for _, branch := range branches {
				branchDict, ok := branch.(SchemaDict)
				if !ok {
					continue
				}
				if branchContradictsParent(root, outer, branchDict) {
					continue
				}
				merged = append(merged, mergeStricter(outer, branchDict))
			}

			for key := range n {
				if key == AnyOf || !distributableKeyword(key, n, root) {
					continue
				}
				delete(n, key)
			}

			if len(merged) == 0 {
				for key := range n {
					delete(n, key)
				}
			} else {
				n[AnyOf] = merged
			}
		}
		for _, value := range n {
			distributeAnyOf(value, root)
		}
	case SchemaList:
		for _, item := range n {
			distributeAnyOf(item, root)
		}
	}
}

// distributableAnyOf returns the branches and the parent constraints to push into
// them, for a node that carries both. Annotations and the root's own $defs / $id
// are not constraints and stay where they are.
func distributableAnyOf(node SchemaDict, root SchemaDict) (SchemaList, SchemaDict, bool) {
	branches, ok := node[AnyOf].(SchemaList)
	if !ok || len(branches) == 0 {
		return nil, nil, false
	}
	for _, branch := range branches {
		if _, ok := branch.(SchemaDict); !ok {
			return nil, nil, false
		}
	}

	outer := make(SchemaDict, len(node))
	for key, value := range node {
		if key != AnyOf && distributableKeyword(key, node, root) {
			outer[key] = value
		}
	}
	if len(outer) == 0 {
		return nil, nil, false
	}

	return branches, outer, true
}

// distributableKeyword reports whether a keyword beside anyOf is a constraint on
// the instance, as opposed to an annotation or a document-level declaration.
func distributableKeyword(keyword string, node SchemaDict, root SchemaDict) bool {
	if CommonKeywords[keyword] {
		return false
	}
	// $defs and $id describe the document, not the instance, and are only legal at
	// the root, which is exactly where they must be left alone.
	return !(TopLevelOnlyKeywords[keyword] && sameSchemaDict(node, root))
}

func sameSchemaDict(a, b SchemaDict) bool {
	return reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer()
}

// branchContradictsParent reports whether no instance can satisfy the branch and
// the parent constraints at once. A branch that only points at a definition is
// resolved first, so that a contradiction hidden behind $ref still counts.
func branchContradictsParent(root SchemaDict, outer SchemaDict, branch SchemaDict) bool {
	if unsatisfiableOverlap(outer, branch) != "" {
		return true
	}

	ref, ok := branch[Ref].(string)
	if !ok || refIsRecursive(root, ref) {
		return false
	}
	target, ok := resolveDefsRef(root, ref)
	if !ok {
		return false
	}
	return unsatisfiableOverlap(outer, target) != ""
}

// pruneOrphanDefs removes root-level $defs entries that nothing points at any
// more. Inlining leaves such entries behind and they only eat into the size
// budget, which a large schema cannot afford to waste. Removing one entry can
// orphan another, so this runs until it settles.
func pruneOrphanDefs(root SchemaDict) {
	for pruneOrphanDefsOnce(root) {
	}
}

func pruneOrphanDefsOnce(root SchemaDict) bool {
	defs, ok := root[Defs].(SchemaDict)
	if !ok || len(defs) == 0 {
		return false
	}

	referenced := make(map[string]bool, len(defs))
	for _, ref := range collectRefs(root) {
		// "#" resolves to the whole document, so nothing is provably unused.
		if ref == "#" {
			return false
		}
		parts, ok := defsRefParts(ref)
		if !ok || len(parts) < 2 || parts[0] != Defs {
			continue
		}
		referenced[parts[1]] = true
	}

	removed := false
	for name := range defs {
		if !referenced[name] {
			delete(defs, name)
			removed = true
		}
	}

	if len(defs) == 0 {
		delete(root, Defs)
	}
	return removed
}

// hasMergeableRefSibling reports whether the subtree holds a node worth merging,
// so the common case costs a single read-only walk.
func hasMergeableRefSibling(node any, root SchemaDict) bool {
	switch n := node.(type) {
	case SchemaDict:
		if _, _, ok := mergeableRefTarget(n, root); ok {
			return true
		}
		for _, value := range n {
			if hasMergeableRefSibling(value, root) {
				return true
			}
		}
	case SchemaList:
		for _, item := range n {
			if hasMergeableRefSibling(item, root) {
				return true
			}
		}
	}
	return false
}

// mergeRefSiblings walks the tree and merges every mergeable node in place. A
// node the budget refuses keeps its $ref and siblings; stripRefSiblings drops
// them afterwards in one pass, the same fallback validation would apply.
func mergeRefSiblings(node any, root SchemaDict, budget *refInlineBudget) {
	switch n := node.(type) {
	case SchemaDict:
		if target, siblings, ok := mergeableRefTarget(n, root); ok {
			// Charge the marginal growth: the merged node replaces the $ref node,
			// so the node's own bytes come back. Merged output never exceeds
			// target plus siblings, so this stays an upper bound of the real cost.
			cost := serializedSize(target) + serializedSize(siblings) - serializedSize(n)
			if cost < 0 {
				cost = 0
			}
			if budget.tryConsume(cost) {
				merged := mergeStricter(siblings, target)
				for key := range n {
					delete(n, key)
				}
				for key, value := range merged {
					n[key] = value
				}
				// The merged result can itself carry a $ref from the definition body,
				// but its siblings have already been folded in, so keep walking the
				// children only.
			}
		}
		for _, value := range n {
			mergeRefSiblings(value, root, budget)
		}
	case SchemaList:
		for _, item := range n {
			mergeRefSiblings(item, root, budget)
		}
	}
}

// mergeableRefTarget returns the referenced schema and the sibling constraints
// when node is a $ref carrying its own constraints that can safely be folded in.
func mergeableRefTarget(node SchemaDict, root SchemaDict) (SchemaDict, SchemaDict, bool) {
	ref, ok := node[Ref].(string)
	if !ok || len(node) < 2 {
		return nil, nil, false
	}

	siblings := make(SchemaDict, len(node)-1)
	for key, value := range node {
		if key != Ref {
			siblings[key] = value
		}
	}
	if len(siblings) == 0 {
		return nil, nil, false
	}

	target, ok := resolveDefsRef(root, ref)
	if !ok || len(target) == 0 {
		return nil, nil, false
	}

	// Only step in when a real constraint would otherwise be lost: the definition
	// either does not carry the keyword at all, or carries a different value.
	// Annotations and exact duplicates are left to the ordinary simplify path,
	// which strips the outer copy without changing what the schema accepts.
	if !siblingsWouldLoseConstraint(siblings, target) {
		return nil, nil, false
	}

	// A contradiction is not something merging can fix; leave the node intact so
	// that validation reports it as unsatisfiable.
	if unsatisfiableOverlap(siblings, target) != "" {
		return nil, nil, false
	}

	if refIsRecursive(root, ref) {
		return nil, nil, false
	}

	return target, siblings, true
}

// mergeRefSiblingDroppingContradiction folds the siblings of node into its $ref
// target for the nodes mergeableRefTarget refuses to touch because the two sides
// contradict each other. Whatever they still agree on survives and only the
// contradicting keyword goes, which leaves the node unconstrained in that one
// dimension instead of asserting one of the two sides.
//
// Two sides that cannot agree on the type are the exception. Every other keyword
// only means anything relative to a type, so combining a string bound from one
// side with a numeric bound from the other produces a schema that describes
// nothing; there the caller is told to keep none of it.
func mergeRefSiblingDroppingContradiction(root, node SchemaDict) (SchemaDict, bool) {
	ref, ok := node[Ref].(string)
	if !ok {
		return nil, false
	}

	target, ok := resolveDefsRef(root, ref)
	if !ok || len(target) == 0 || refIsRecursive(root, ref) {
		return nil, false
	}

	siblings := make(SchemaDict, len(node)-1)
	for key, value := range node {
		if key != Ref {
			siblings[key] = value
		}
	}
	if len(siblings) == 0 {
		return nil, false
	}

	if typeSetsDisjoint(typeSet(siblings[Type]), typeSet(target[Type])) {
		return nil, false
	}

	return mergeStricter(siblings, target), true
}

// siblingsWouldLoseConstraint reports whether dropping the siblings would weaken
// the schema.
func siblingsWouldLoseConstraint(siblings, target SchemaDict) bool {
	for key, value := range siblings {
		if CommonKeywords[key] {
			continue
		}
		existing, ok := target[key]
		if !ok || !reflect.DeepEqual(existing, value) {
			return true
		}
	}
	return false
}

// resolveDefsRef resolves "#/$defs/Name" style pointers against root. "#" is
// deliberately unsupported: it always resolves to the whole document and is
// therefore recursive by construction.
func resolveDefsRef(root SchemaDict, ref string) (SchemaDict, bool) {
	parts, ok := defsRefParts(ref)
	if !ok {
		return nil, false
	}

	current := root
	for _, part := range parts {
		next, ok := current[part].(SchemaDict)
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

// defsRefParts splits "#/$defs/A/$defs/B" into its dictionary keys.
func defsRefParts(ref string) ([]string, bool) {
	const prefix = "#/"
	if len(ref) <= len(prefix) || ref[:len(prefix)] != prefix {
		return nil, false
	}

	var parts []string
	start := len(prefix)
	for i := start; i <= len(ref); i++ {
		if i == len(ref) || ref[i] == '/' {
			if i == start {
				return nil, false
			}
			parts = append(parts, ref[start:i])
			start = i + 1
		}
	}
	if len(parts) == 0 {
		return nil, false
	}
	return parts, true
}

// refIsRecursive reports whether following ref can lead back to ref itself,
// directly or through other definitions.
func refIsRecursive(root SchemaDict, ref string) bool {
	seen := map[string]bool{ref: true}
	queue := []string{ref}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		target, ok := resolveDefsRef(root, current)
		if !ok {
			continue
		}

		for _, next := range collectRefs(target) {
			if next == ref {
				return true
			}
			// "#" pulls in the whole document, definitions included, so treat any
			// definition that reaches it as recursive.
			if next == "#" {
				return true
			}
			if !seen[next] {
				seen[next] = true
				queue = append(queue, next)
			}
		}
	}
	return false
}

// collectRefs gathers every $ref string in the subtree.
func collectRefs(node any) []string {
	var refs []string
	switch n := node.(type) {
	case SchemaDict:
		if ref, ok := n[Ref].(string); ok {
			refs = append(refs, ref)
		}
		for _, value := range n {
			refs = append(refs, collectRefs(value)...)
		}
	case SchemaList:
		for _, item := range n {
			refs = append(refs, collectRefs(item)...)
		}
	}
	return refs
}

// mergeStricter folds the sibling constraints into a copy of the referenced
// definition, keeping the stricter value wherever both sides set a keyword.
func mergeStricter(siblings, target SchemaDict) SchemaDict {
	merged := make(SchemaDict, len(siblings)+len(target))
	for key, value := range target {
		merged[key] = deepCopyValue(value)
	}

	for key, value := range siblings {
		existing, clash := merged[key]
		if !clash {
			merged[key] = deepCopyValue(value)
			continue
		}
		if stricter, keep := stricterValue(key, existing, deepCopyValue(value)); keep {
			merged[key] = stricter
		} else {
			delete(merged, key)
		}
	}

	return merged
}

// stricterValue picks whichever of the two values admits fewer instances.
// fromTarget comes from the definition, fromSibling from the use site. A false
// second return means the two sides have nothing in common: there is no value to
// keep, so the caller drops the keyword rather than picking a side and accepting
// what the other side ruled out.
func stricterValue(keyword string, fromTarget, fromSibling any) (any, bool) {
	switch {
	case lowerBoundKeywords[keyword]:
		return largerNumber(fromTarget, fromSibling), true
	case upperBoundKeywords[keyword]:
		return smallerNumber(fromTarget, fromSibling), true
	case keyword == Type:
		return intersectTypes(fromTarget, fromSibling)
	case keyword == Enum:
		return intersectEnums(fromTarget, fromSibling)
	case keyword == Required:
		return unionRequired(fromTarget, fromSibling), true
	case keyword == Properties:
		return mergeProperties(fromTarget, fromSibling), true
	}

	// Anything else, annotations included, keeps the use site's value: it is the
	// more specific of the two. Two different patterns cannot be intersected into
	// one pattern, so the same rule applies there.
	return fromSibling, true
}

func largerNumber(a, b any) any {
	x, okA := numericValue(a)
	y, okB := numericValue(b)
	if !okA {
		return b
	}
	if !okB {
		return a
	}
	if x >= y {
		return a
	}
	return b
}

func smallerNumber(a, b any) any {
	x, okA := numericValue(a)
	y, okB := numericValue(b)
	if !okA {
		return b
	}
	if !okB {
		return a
	}
	if x <= y {
		return a
	}
	return b
}

func numericValue(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	}
	return 0, false
}

// intersectTypes keeps the types allowed by both sides. integer wins over number
// because every integer is a number but not the other way round.
func intersectTypes(fromTarget, fromSibling any) (any, bool) {
	setTarget, setSibling := typeSet(fromTarget), typeSet(fromSibling)
	if len(setTarget) == 0 {
		return fromSibling, true
	}
	if len(setSibling) == 0 {
		return fromTarget, true
	}

	var names []string
	for name := range setTarget {
		if _, ok := setSibling[name]; ok {
			names = append(names, name)
			continue
		}
		// integer is the narrower side of an integer/number overlap
		switch name {
		case Number:
			if _, ok := setSibling[Integer]; ok {
				names = append(names, Integer)
			}
		case Integer:
			if _, ok := setSibling[Number]; ok {
				names = append(names, Integer)
			}
		}
	}

	if len(names) == 0 {
		return nil, false
	}

	sort.Strings(names)
	if len(names) == 1 {
		return names[0], true
	}
	list := make(SchemaList, len(names))
	for i, name := range names {
		list[i] = name
	}
	return list, true
}

// intersectEnums keeps the values allowed by both sides.
func intersectEnums(fromTarget, fromSibling any) (any, bool) {
	listTarget, okTarget := fromTarget.(SchemaList)
	listSibling, okSibling := fromSibling.(SchemaList)
	if !okTarget || !okSibling {
		return fromSibling, true
	}

	var out SchemaList
	for _, candidate := range listTarget {
		for _, allowed := range listSibling {
			if reflect.DeepEqual(candidate, allowed) {
				out = append(out, candidate)
				break
			}
		}
	}

	if len(out) == 0 {
		return nil, false
	}
	return out, true
}

// unionRequired keeps every property either side insists on.
func unionRequired(fromTarget, fromSibling any) any {
	listTarget, _ := fromTarget.(SchemaList)
	listSibling, _ := fromSibling.(SchemaList)

	seen := make(map[string]bool, len(listTarget)+len(listSibling))
	out := make(SchemaList, 0, len(listTarget)+len(listSibling))
	for _, list := range []SchemaList{listTarget, listSibling} {
		for _, item := range list {
			name, ok := item.(string)
			if !ok || seen[name] {
				continue
			}
			seen[name] = true
			out = append(out, name)
		}
	}

	if len(out) == 0 {
		return fromSibling
	}
	return out
}

// mergeProperties keeps properties from both sides. A property both sides
// describe is merged the same way as any other subschema, so neither side's
// constraints are dropped; letting the use site simply win would lose whatever
// the definition asserted about that property.
func mergeProperties(fromTarget, fromSibling any) any {
	dictTarget, okTarget := fromTarget.(SchemaDict)
	dictSibling, okSibling := fromSibling.(SchemaDict)
	if !okTarget || !okSibling {
		return fromSibling
	}

	out := make(SchemaDict, len(dictTarget)+len(dictSibling))
	for name, value := range dictTarget {
		out[name] = value
	}
	for name, value := range dictSibling {
		existing, clash := out[name]
		if !clash {
			out[name] = value
			continue
		}

		targetProperty, okT := existing.(SchemaDict)
		siblingProperty, okS := value.(SchemaDict)
		if !okT || !okS {
			out[name] = value
			continue
		}
		out[name] = mergeStricter(siblingProperty, targetProperty)
	}
	return out
}
