"use client";

import { useCallback, useEffect, useState, type ComponentType } from "react";
import { errorMessage } from "@/lib/api";
import { ErrorMessage } from "@/components/ErrorMessage";
import { Spinner } from "@/components/Spinner";

export interface AsyncDataProps<T> {
  data: T;
  reload: () => void;
}

type State<T> =
  | { status: "loading" }
  | { status: "error"; message: string }
  | { status: "ready"; data: T };

/**
 * HoC: loads data with `load(props)` and renders the wrapped component only
 * once it is available, handling the loading and error states uniformly.
 */
export function withAsyncData<P extends object, T>(load: (props: P) => Promise<T>) {
  return function wrap(Wrapped: ComponentType<P & AsyncDataProps<T>>): ComponentType<P> {
    function WithAsyncData(props: P) {
      const [state, setState] = useState<State<T>>({ status: "loading" });
      const [version, setVersion] = useState(0);
      const reload = useCallback(() => setVersion((v) => v + 1), []);

      useEffect(() => {
        let cancelled = false;
        load(props)
          .then((data) => !cancelled && setState({ status: "ready", data }))
          .catch((err: unknown) => !cancelled && setState({ status: "error", message: errorMessage(err) }));
        return () => {
          cancelled = true;
        };
        // Reload when any prop value changes or reload() is called.
      }, [version, ...Object.values(props)]);

      if (state.status === "loading") return <Spinner />;
      if (state.status === "error") {
        return (
          <ErrorMessage
            message={state.message}
            onRetry={() => {
              setState({ status: "loading" });
              reload();
            }}
          />
        );
      }
      return <Wrapped {...props} data={state.data} reload={reload} />;
    }
    WithAsyncData.displayName = `withAsyncData(${Wrapped.displayName ?? Wrapped.name ?? "Component"})`;
    return WithAsyncData;
  };
}
