export const manifest = (() => {
function __memo(fn) {
	let value;
	return () => value ??= (value = fn());
}

return {
	appDir: "_app",
	appPath: "chat/_app",
	assets: new Set(["favicon.svg"]),
	mimeTypes: {".svg":"image/svg+xml"},
	_: {
		client: {start:"_app/immutable/entry/start.Dq6QS7Ma.js",app:"_app/immutable/entry/app.CG9-DsT6.js",imports:["_app/immutable/entry/start.Dq6QS7Ma.js","_app/immutable/chunks/BdGEvIS4.js","_app/immutable/chunks/BAg31RI9.js","_app/immutable/chunks/B-oOwpVJ.js","_app/immutable/entry/app.CG9-DsT6.js","_app/immutable/chunks/BAg31RI9.js","_app/immutable/chunks/226DXbFe.js","_app/immutable/chunks/BwfNe992.js","_app/immutable/chunks/B-oOwpVJ.js","_app/immutable/chunks/BWarOLcd.js","_app/immutable/chunks/CuksnDLX.js","_app/immutable/chunks/DTS05npu.js"],stylesheets:[],fonts:[],uses_env_dynamic_public:false},
		nodes: [
			__memo(() => import('./nodes/0.js')),
			__memo(() => import('./nodes/1.js')),
			__memo(() => import('./nodes/2.js')),
			__memo(() => import('./nodes/3.js'))
		],
		remotes: {
			
		},
		routes: [
			{
				id: "/",
				pattern: /^\/$/,
				params: [],
				page: { layouts: [0,], errors: [1,], leaf: 2 },
				endpoint: null
			},
			{
				id: "/login",
				pattern: /^\/login\/?$/,
				params: [],
				page: { layouts: [0,], errors: [1,], leaf: 3 },
				endpoint: null
			}
		],
		prerendered_routes: new Set([]),
		matchers: async () => {
			
			return {  };
		},
		server_assets: {}
	}
}
})();
