export function EventsOn(eventName, callback) {
  return window.runtime.EventsOn(eventName, callback)
}

export function EventsOff(eventName, ...additionalEventNames) {
  window.runtime.EventsOff(eventName, ...additionalEventNames)
}
