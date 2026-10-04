// Set by the build from git (Makefile, Dockerfile); a plain `vite` run has neither.
declare global {
  interface ImportMetaEnv {
    readonly VITE_VERSION?: string;
    readonly VITE_COMMIT?: string;
  }
}

export const VERSION = import.meta.env.VITE_VERSION || "dev";
export const COMMIT = import.meta.env.VITE_COMMIT || "unknown";

export const REPOSITORY = "https://github.com/raphalogou/homebase";
