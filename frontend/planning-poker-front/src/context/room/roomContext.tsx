'use client'

import React, { createContext, RefObject, useContext, useRef } from 'react';

type RoomContextType = {
  socket: RefObject<WebSocket | null>;
  connected: RefObject<boolean>;
};

const RoomContext = createContext<RoomContextType | null>(null);

export function RoomProvider({ children }: Readonly<{ children: React.ReactNode }>) {
  const socket = useRef<WebSocket | null>(null);
  const connected = useRef(false);

  const value = React.useMemo(() => ({ socket, connected }), [socket, connected]);

  return (
    <RoomContext.Provider value={value}>
      {children}
    </RoomContext.Provider>
  );
}

export function useRoom() {
  const context = useContext(RoomContext);
  if (!context) {
    throw new Error('useRoom must be used within RoomProvider');
  }
  return context;
}
