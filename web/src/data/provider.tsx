import {
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import { ApiError, api } from "./api.ts";
import { openDatabase } from "./idb.ts";
import { LocalStore } from "./store.ts";
import { SyncEngine } from "./sync.ts";

export type AuthState = "loading" | "signedOut" | "signedIn";

interface DataContext {
  store: LocalStore;
  engine: SyncEngine;
  auth: AuthState;
  login: (passphrase: string) => Promise<void>;
  logout: () => Promise<void>;
}

const Context = createContext<DataContext | null>(null);

interface Ready {
  store: LocalStore;
  engine: SyncEngine;
}

/**
 * Opens the local database and decides whether the user is signed in. A
 * device that signed in before opens straight into the app, even offline;
 * the next sync finds out if the session has ended.
 */
export function DataProvider({ children, fallback }: { children: ReactNode; fallback: ReactNode }) {
  const [ready, setReady] = useState<Ready | null>(null);
  const [auth, setAuth] = useState<AuthState>("loading");

  useEffect(() => {
    let cancelled = false;
    let engine: SyncEngine | null = null;

    (async () => {
      const db = await openDatabase();
      const store = new LocalStore(db);
      await store.load();
      engine = new SyncEngine(store, api, () => {
        void store.setSignedIn(false);
        setAuth("signedOut");
      });
      if (cancelled) return;
      setReady({ store, engine });

      if (store.signedIn) {
        setAuth("signedIn");
        engine.start();
        api.me().then(
          (me) => void store.setMe(me),
          () => {},
        );
        return;
      }
      try {
        const me = await api.me();
        await store.setMe(me);
        await store.setSignedIn(true);
        if (cancelled) return;
        setAuth("signedIn");
        engine.start();
      } catch {
        if (!cancelled) setAuth("signedOut");
      }
    })();

    return () => {
      cancelled = true;
      engine?.stop();
    };
  }, []);

  const login = useCallback(
    async (passphrase: string) => {
      if (!ready) return;
      await api.login(passphrase);
      const me = await api.me();
      await ready.store.setMe(me);
      await ready.store.setSignedIn(true);
      setAuth("signedIn");
      ready.engine.stop();
      ready.engine.start();
    },
    [ready],
  );

  const logout = useCallback(async () => {
    if (!ready) return;
    // Send what is still waiting, so logging out does not lose it.
    await ready.engine.run();
    ready.engine.stop();
    try {
      await api.logout();
    } catch (err) {
      if (err instanceof ApiError && err.status !== 401) throw err;
    }
    await ready.store.reset();
    setAuth("signedOut");
  }, [ready]);

  const value = useMemo(
    () => (ready ? { ...ready, auth, login, logout } : null),
    [ready, auth, login, logout],
  );

  if (!value || auth === "loading") return <>{fallback}</>;
  return <Context.Provider value={value}>{children}</Context.Provider>;
}

export function useData(): DataContext {
  const ctx = useContext(Context);
  if (!ctx) throw new Error("useData must be used inside DataProvider");
  return ctx;
}
