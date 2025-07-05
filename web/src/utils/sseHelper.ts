import { API_ENDPOINTS } from '../config'; // Make sure this path is correct

interface EventHandlers {
  [eventName: string]: (data: any) => void;
}

interface SSEHelperOptions {
  onOpen?: (event: Event) => void;
  onError?: (event: Event) => void;
  onDefaultMessage?: (event: MessageEvent) => void; // For messages without a specific event type
}

/**
 * Initializes an EventSource connection and attaches event handlers.
 *
 * @param url The URL for the SSE endpoint.
 * @param eventHandlers A map where keys are event names and values are handler functions.
 * @param options Optional callbacks for onopen, onerror, and onDefaultMessage.
 * @returns A cleanup function to close the EventSource.
 */
export const initializeSSE = (
  url: string,
  eventHandlers: EventHandlers,
  options?: SSEHelperOptions
): (() => void) => {
  const eventSource = new EventSource(url, { withCredentials: true }); // withCredentials might be needed depending on auth

  eventSource.onopen = (event) => {
    console.log('SSE Connection Opened:', event);
    if (options?.onOpen) {
      options.onOpen(event);
    }
  };

  eventSource.onerror = (event) => {
    console.error('SSE Error:', event);
    if (options?.onError) {
      options.onError(event);
    }
    // Note: EventSource will automatically try to reconnect on most errors.
    // If the server closes the connection with a 204 or specific error, it might not.
    // Consider specific error handling like closing if error indicates a permanent failure.
  };

  // Handle messages that don't have a specific event type
  if (options?.onDefaultMessage) {
    eventSource.onmessage = options.onDefaultMessage;
  }

  // Attach custom event listeners
  Object.entries(eventHandlers).forEach(([eventName, handler]) => {
    eventSource.addEventListener(eventName, (event: MessageEvent) => {
      try {
        const data = JSON.parse(event.data);
        handler(data);
      } catch (e) {
        console.error(`Error parsing SSE event data for event "${eventName}":`, e, event.data);
      }
    });
  });

  // Cleanup function
  return () => {
    console.log('Closing SSE Connection.');
    eventSource.close();
    // It's good practice to also remove event listeners if they were added dynamically,
    // but EventSource.close() should handle most cleanup.
    // For addEventListener, if we had stored references to the wrapped handlers, we could remove them.
    // However, for this basic helper, close() is the primary cleanup.
  };
};

// Example Usage (for demonstration, not to be run here):
/*
const cleanup = initializeSSE(
  API_ENDPOINTS.sseStatsStream,
  {
    'summary_update': (data) => {
      console.log('Received summary_update:', data);
      // Update React state here
    },
    'queue_update': (data) => {
      console.log('Received queue_update:', data);
      // Update React state here
    }
  },
  {
    onError: (event) => console.error("Custom SSE error handler:", event)
  }
);

// Later, in a useEffect cleanup or when the component unmounts:
// cleanup();
*/
