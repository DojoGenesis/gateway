

export const index = 0;
let component_cache;
export const component = async () => component_cache ??= (await import('../entries/pages/_layout.svelte.js')).default;
export const universal = {
  "ssr": false,
  "prerender": false
};
export const universal_id = "src/routes/+layout.ts";
export const imports = ["_app/immutable/nodes/0.DTRmNCkC.js","_app/immutable/chunks/BwfNe992.js","_app/immutable/chunks/BAg31RI9.js","_app/immutable/chunks/CuksnDLX.js","_app/immutable/chunks/DwVGEHRN.js"];
export const stylesheets = ["_app/immutable/assets/0.-LOnyrt6.css"];
export const fonts = [];
