Production-Level Prompt for AI: Radial Skill Tree Microfrontend
Project Overview
Generate a Vite + React SPA. Do NOT use Next.js or any SSR features. Use Zustand for state management and Tailwind CSS for styling. The app must fetch data from a Go REST API. Implement the Profile Skill Tree as an SVG component based on the provided math logic.

Math/Geometry: SVG Paths (NO Canvas, NO D3-chart libraries, but helper math functions are encouraged)

Core Concept: Radial Skill Tree (Sunburst)
Concentric Rings: Represent depth/level (Level 1 in the center, Level N at the edge).

Segments: Represent specific skills.

Hierarchy: Each skill (except Level 1) has a parentId.

Growth Metaphor: Skills grow outward. A child skill is "Locked" if its parent hasn't reached 70% progress.

Visualization & Geometry Logic
1. Dynamic Layout Calculation
Instead of hardcoded angles, implement a utility that:

Groups skills by top-level categories.

Distributes the 360° circle equally among categories.

Calculates angleStart and angleEnd for child nodes based on their parent's angular span.

2. SVG Rendering Requirements
Use a single <svg viewBox="-100 -100 200 200"> for easy scaling.

Implement describeArc(x, y, radius, startAngle, endAngle) to generate d attributes for <path />.

Padding: Add a 0.02rad gap between segments for a "clean" look.

Corner Radius: If possible, implement slight rounding for segment edges.

Data Model & Types
TypeScript
export type SkillNode = {
  id: string;
  title: string;
  category: string;
  level: number;       // 1 = innermost, 4 = outermost
  progress: number;    // 0 to 1.0
  parentId?: string;
  metadata?: Record<string, any>;
};

export type SkillGeometry = SkillNode & {
  startAngle: number;
  endAngle: number;
  innerRadius: number;
  outerRadius: number;
  isLocked: boolean;
};
Component Requirements
1. SkillRadialChart (Core)
Calculate geometry in a useMemo hook based on the raw skills array.

Color Logic: Map progress to Tailwind tokens:

0.0–0.2: bg-slate-200 (Gray)

0.2–0.5: bg-yellow-400 (Beginner)

0.5–0.8: bg-emerald-500 (Confident)

0.8–1.0: bg-indigo-600 (Mastery)

Lock State: If isLocked, render with opacity-30 and a "lock" icon overlay.

2. Zustand Store (State)
skills: SkillNode[]

selectedSkillId: string | null

hoveredSkillId: string | null

Actions: setHover(id), selectSkill(id).

3. SkillDetailsPanel (UI)
Use a shadcn/ui Sheet (Drawer) or Dialog.

Show skill title, category, a detailed progress bar, and a "Start Training" button.

Interactions & UX
Hover: Slightly scale the segment (via CSS transform or radius offset) and show a shadcn Tooltip.

Click: Only trigger selectSkill if the node is NOT locked.

Responsive: The SVG must be wrapped in a container that maintains aspect ratio.

Output Requirements
Code Structure: Separate files for types.ts, math-utils.ts, store.ts, and components.

Implementation: Full TypeScript code with comments explaining the polar-to-cartesian conversion.

Mock Data: Create a robust set of 15+ skills across 3 categories (e.g., "Algebra", "Geometry", "Calculus") to demonstrate the tree growth.

No Junk: No unused imports, no any types, no external heavy libraries.