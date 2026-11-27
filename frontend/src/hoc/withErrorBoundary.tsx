"use client";

import { Component, type ComponentType, type ErrorInfo, type ReactNode } from "react";
import { ErrorMessage } from "@/components/ErrorMessage";

interface State {
  error: Error | null;
}

class ErrorBoundary extends Component<{ children: ReactNode }, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("Unhandled UI error", error, info.componentStack);
  }

  render() {
    if (this.state.error) {
      return (
        <ErrorMessage
          message="Algo salió mal al mostrar esta vista."
          onRetry={() => this.setState({ error: null })}
        />
      );
    }
    return this.props.children;
  }
}

/** HoC: isolates render errors of the wrapped component behind a fallback UI. */
export function withErrorBoundary<P extends object>(Wrapped: ComponentType<P>): ComponentType<P> {
  function WithErrorBoundary(props: P) {
    return (
      <ErrorBoundary>
        <Wrapped {...props} />
      </ErrorBoundary>
    );
  }
  WithErrorBoundary.displayName = `withErrorBoundary(${Wrapped.displayName ?? Wrapped.name ?? "Component"})`;
  return WithErrorBoundary;
}
