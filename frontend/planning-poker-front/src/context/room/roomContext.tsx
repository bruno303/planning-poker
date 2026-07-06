'use client'

import React, { createContext, RefObject, useContext, useRef } from 'react';

type RoomContextType = {
  socket: RefObject<WebSocket | null>;
  connected: RefObject<boolean>;
};

const RoomContext = createContext<RoomContextType | null>(null);

export function RoomProvider({ children }: { children: React.ReactNode }) {
  const socket = useRef<WebSocket | null>(null);
  const connected = useRef(false);

  return (
    <RoomContext.Provider value={{ socket, connected }}>
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
