import {Injectable} from '@angular/core';
import {HttpClient, HttpHeaders} from "@angular/common/http";
import {Observable} from "rxjs";
import {MCServerDetail} from "../../model/serverDetail";

@Injectable({
  providedIn: 'root'
})
export class RestService {

  private headers: HttpHeaders = new HttpHeaders({'Content-Type': 'application/json'})
  private baseUrl: string = "http://localhost:3000/api/v1"


  constructor(private http: HttpClient) {

  }


  createNewServer(name: string): Observable<any> {
    return this.http.post<any>(this.baseUrl + "/container/create",
      JSON.stringify({servername: name}),
      {headers: this.headers}
    )
  }

  getAllServersWithDetails(): Observable<MCServerDetail[]> {
    return this.http.get<MCServerDetail[]>(this.baseUrl + "/container/withDetails")
  }

  stopContainer(containerId: string): Observable<MCServerDetail> {
    return this.http.get<MCServerDetail>(this.baseUrl + "/container/stop/" + containerId)
  }

  startContainer(containerId: string): Observable<MCServerDetail> {
    return this.http.get<MCServerDetail>(this.baseUrl + "/container/start/" + containerId)
  }

  restartContainer(containerId: string): Observable<MCServerDetail> {
    return this.http.get<MCServerDetail>(this.baseUrl + "/container/restart/" + containerId)
  }

  deleteContainer(containerId: string): Observable<any> {
    return this.http.delete<any>(this.baseUrl + "/container/id/" + containerId)
  }

}
