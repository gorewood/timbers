---
name: Timbermill
description: A contemporary working mill for durable development reasoning.
colors:
  mill-black: "#111514"
  graphite: "#1c2422"
  mineral: "#f3f5f1"
  paper: "#fbfcf9"
  safety-yellow: "#f2c230"
  oxidized-teal: "#2b7a78"
  signal-coral: "#d85f45"
  line-dark: "#33413d"
  line-light: "#d9dfda"
typography:
  display:
    fontFamily: "Geologica, system-ui, sans-serif"
    fontSize: "6rem"
    fontWeight: 760
    lineHeight: 0.94
    letterSpacing: "0"
  body:
    fontFamily: "Geologica, system-ui, sans-serif"
    fontSize: "1rem"
    fontWeight: 420
    lineHeight: 1.7
    letterSpacing: "0"
  mono:
    fontFamily: "ui-monospace, SFMono-Regular, Consolas, monospace"
    fontSize: "0.875rem"
    fontWeight: 500
    lineHeight: 1.55
    letterSpacing: "0"
rounded:
  sm: "4px"
  md: "8px"
spacing:
  sm: "8px"
  md: "16px"
  lg: "32px"
  xl: "64px"
components:
  button-primary:
    backgroundColor: "{colors.safety-yellow}"
    textColor: "{colors.mill-black}"
    rounded: "{rounded.sm}"
    padding: "12px 18px"
  button-secondary:
    backgroundColor: "{colors.graphite}"
    textColor: "{colors.paper}"
    rounded: "{rounded.sm}"
    padding: "12px 18px"
---

# Design System: Timbermill

## 1. Overview

**Creative North Star: "The Working Mill"**

Timbermill presents development history as material moving through a precise contemporary mill. Raw commits enter, reasoning is captured and graded, and durable artifacts leave. The visual system combines dark machinery, pale reading surfaces, safety markings, real cut material, and restrained operational signals.

The homepage may be cinematic; collections and reports become a quiet reading floor. The system rejects rustic nostalgia, generic developer dark mode, editorial affectation, and repeated card scaffolds.

**Key Characteristics:**
- High-contrast industrial surfaces with a varied functional palette
- One decisive material image instead of decorative stock collections
- Real ledger and report output used as the primary visual proof
- Dense, precise metadata paired with generous long-form reading space
- Motion concentrated around transformation and navigation

## 2. Colors

Graphite machinery and mineral reading surfaces are punctuated by safety yellow, oxidized teal, and sparing signal coral.

### Primary
- **Safety Mark** (`#f2c230`): Primary actions, active stages, focus emphasis, and the central Timbers promise.

### Secondary
- **Oxidized Teal** (`#2b7a78`): Links, informational states, and navigation feedback.

### Tertiary
- **Signal Coral** (`#d85f45`): Rare urgency, selected artifact accents, and meaningful contrast against teal and yellow.

### Neutral
- **Mill Black** (`#111514`): Hero and footer foundations.
- **Graphite** (`#1c2422`): Elevated dark surfaces and code.
- **Mineral** (`#f3f5f1`): Section bands and secondary reading surfaces.
- **Paper** (`#fbfcf9`): Primary report background.
- **Line Dark / Light** (`#33413d`, `#d9dfda`): Structural dividers appropriate to each surface.

**The Safety Mark Rule.** Yellow marks decisions and forward movement; it is not a general background color.

## 3. Typography

**Display Font:** Geologica (with system-ui fallback)
**Body Font:** Geologica (with system-ui fallback)
**Label/Mono Font:** Native platform monospace

**Character:** Geologica's variable geometry supplies both mechanical precision and subtle organic movement. Monospace is reserved for commands, metadata, and captured ledger structure rather than used as shorthand for the whole technical brand.

### Hierarchy
- **Display** (760, `6rem`, 0.94): Product name and one major collection title, stepped down at explicit breakpoints.
- **Headline** (700, `4.5rem`, 1.02): Section-defining claims, stepped down at explicit breakpoints.
- **Title** (650, `2rem`, 1.15): Artifact and stage titles, stepped down at explicit breakpoints.
- **Body** (420, `1rem`, 1.7): Reading copy capped at 70 characters.
- **Label** (650, `0.75rem`, 0, normal case): Operational metadata and short status labels.

**The Two Registers Rule.** Brand pages use compressed scale and dramatic grouping; report prose uses measured scale and generous leading.

## 4. Elevation

The system is flat by default. Depth comes from tonal layering, image overlap, and structural rules. Compact shadows appear only on transient navigation and interactive elements that physically rise on hover; decorative ambient shadows are excluded.

**The Flat Machinery Rule.** Surfaces sit flush until interaction gives elevation a purpose.

## 5. Components

### Buttons
- **Shape:** Compact working control with a 4px radius.
- **Primary:** Safety yellow on mill black with 12px by 18px padding.
- **Hover / Focus:** Small directional movement, stronger ink contrast, and a visible focus ring.
- **Secondary:** Transparent or graphite treatment with a structural border, never a ghost-card shadow.

### Cards / Containers
- **Corner Style:** 8px maximum.
- **Background:** Paper or mineral for reading; graphite for ledger proof.
- **Shadow Strategy:** None at rest.
- **Border:** Structural dividers or a full quiet outline; no colored side stripes.
- **Internal Padding:** 24px to 32px according to density.

### Navigation
- The custom Timbermill mark and wordmark anchor the header. Familiar actions use Lucide symbols with text where ambiguity remains. Desktop navigation is restrained; mobile navigation uses an icon button with a clear accessible label.

### Install Rail
- A dark, full-width operational strip integrates the command, shell prompt, and copy action without resembling a floating code card.

### Artifact Row
- Reports appear as varied horizontal records with type, date, summary, and a directional cue. Rows reveal structure on hover without changing dimensions.

## 6. Do's and Don'ts

### Do:
- **Do** show raw commits becoming captured reasoning and published artifacts.
- **Do** use real report and ledger content as proof.
- **Do** keep report prose on paper or mineral surfaces with a 65-70 character measure.
- **Do** use one decisive material image with useful composition and accurate alt text.
- **Do** support reduced motion and visible keyboard focus.

### Don't:
- **Don't** make Timbers look rustic, outdoorsy, or nostalgic.
- **Don't** use generic forest photography, fake wood textures, or lumberjack motifs.
- **Don't** use beige editorial minimalism, terminal cosplay, interchangeable SaaS cards, or decorative technical diagrams.
- **Don't** place icons in repeated rounded boxes above headings.
- **Don't** apply entrance motion to every section or hide content until JavaScript runs.
- **Don't** use colored side-stripe borders, gradient text, glassmorphism, or decorative grid backgrounds.
