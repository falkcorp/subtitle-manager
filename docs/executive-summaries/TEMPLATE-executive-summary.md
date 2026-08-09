<!-- file: docs/executive-summaries/TEMPLATE-executive-summary.md -->
<!-- version: 1.0.0 -->
<!-- guid: 4c1e9b73-05a8-42df-9e6a-7b3d20f8c194 -->
<!-- last-edited: 2026-08-09 -->

# Executive summary template

This is the house format for executive summaries in this repository. It exists
so that someone who makes funding and priority decisions — and who does not read
code — can understand what was done, what it cost, what risk it removed, and
what is still unresolved.

**Delete this preamble and everything above the horizontal rule when you copy
the template.**

## How to use it

**Filename:** `YYYY-MM-DD-<slug>-executive-summary.md`, in
`docs/executive-summaries/`. The date is **when the document was written**, not
the period it covers. A monthly roundup written on the 9th covering all of July
is still dated the 9th.

**Two kinds of document:**

| Kind | When | Title style |
| --- | --- | --- |
| **Single-topic** | One incident, defect, or piece of work worth its own write-up | Narrative and specific — "The safety net that was still half on the floor", "Two controls that lied" |
| **Roundup** | A month or a period, grouping many changes into themes | Plain — `Executive Summary: <Period> Roundup` |

**Update in place rather than writing a new one** when the same period or topic
is revisited. Bump `version:` and `last-edited:` in the header. A superseded
claim gets corrected in the text, not silently dropped — see *Honesty rules*.

## The rules that make these documents worth reading

1. **Write for someone non-technical who controls budget.** No jargon without an
   inline gloss: "Pebble (the embedded database the product ships with)",
   "sidecar file (a subtitle stored next to the video rather than inside it)".
2. **Lead with impact, not activity.** "The Media Library page rendered
   completely empty" beats "refactored the browse response handler".
3. **Every claim carries evidence.** Name PR numbers inline. Give real
   measurements. "Verified at roughly 606 books per second across 44,300 books"
   is worth more than "improved performance".
4. **Say what is still broken.** A summary that lists only wins is marketing. The
   `What this means going forward` section is where unproven and unfinished work
   goes, explicitly.
5. **Do not inflate.** If a period was mostly automated dependency updates, say
   so in one line and do not manufacture prose. Padding destroys the credibility
   of the documents that do describe real work.
6. **Correct the record out loud.** If a previous summary said something that
   turned out to be wrong, open by retracting it and explain why it was wrong.
   That convention is what makes the whole series trustworthy.

---

<!-- file: docs/executive-summaries/YYYY-MM-DD-slug-executive-summary.md -->
<!-- version: 1.0.0 -->
<!-- guid: GENERATE-A-NEW-GUID -->
<!-- last-edited: YYYY-MM-DD -->

# Executive Summary: `<Period>` Roundup

**Shipped:** PRs [#AAAA–#BBBB](https://github.com/falkcorp/subtitle-manager/pulls?q=is%3Apr+is%3Amerged+merged%3AYYYY-MM-DD..YYYY-MM-DD),
covering `YYYY-MM-DD` through `YYYY-MM-DD` (`N` merged pull requests).
**Related doc:** `<link to any companion summary, or delete this line>`

`<One or two sentences: is this a roundup grouping many changes by theme, or a
single-topic write-up? What should the reader take away if they read no
further?>`

## Executive Summary

`<One bullet per theme. Each bullet must stand alone — a reader who reads only
this section should understand the period. Name the most important PR numbers
inline as evidence. Order by importance to the business, not chronologically.>`

- **`<Theme name>`.** `<What changed and why it mattered, in plain language.>`
- **`<Theme name>`.** `<...>`
- `<If dependency updates or similar routine work happened, note the count in a
  single bullet and state that it is not broken out below.>`

**Highest-risk items this period** — `<the ones a stakeholder most needs to know
about, because each touched security, money, or could have destroyed data before
it was caught. Delete this block only if there genuinely were none.>`

- **#NNNN** — `<what the risk was, and that it is now closed.>`

## What changed, in plain terms

`<Numbered subsections, in the same order as the Executive Summary bullets. One
per theme. This is where a reader who wants detail goes.>`

### 1. `<Theme name>`

`<Plain-language explanation. Structure that works well:>`

**What was wrong:** `<the user-visible symptom, not the code defect>`

**The fix:** `<what was done, and how it was verified — name the measurement>`

**What it means:** `<consequence for users or for the operator>`

### 2. `<Theme name>`

`<...>`

## What this means going forward

`<Required section. Cover:>`

- `<What is now believed solid, and what evidence supports that.>`
- `<What remains unproven or unfinished — be specific, name it.>`
- `<Any decision now waiting on a human, and what it is blocking.>`
