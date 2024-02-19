import {Injectable} from '@angular/core';
import {io, Socket} from 'socket.io-client';
import {Observable} from "rxjs";


@Injectable({
  providedIn: 'root'
})
export class WebsocketService {

  private socket: Socket;

  constructor() {
    this.socket = io("ws://localhost:3000", {'path': '/ws/v1/'})
    console.log(this.socket.io)
  }

  connect(): void {
    this.socket.connect()
  }

  disconnect() {
    this.socket.disconnect()
  }

  sendMessage(message: any) {
    this.socket.emit('message', message)
  }

  receiveMessage(): Observable<string> {
    return new Observable<string>((observer) => {
      this.socket.on('message', (data: string) => {
        observer.next(data)
      })
    })
  }


}
