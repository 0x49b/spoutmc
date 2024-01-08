import {Component, Input, OnInit, ViewChild, ViewContainerRef} from '@angular/core';
import {Plugin} from "../../../model/plugins";
import {ClrTreeNode, ClrTreeViewModule} from "@clr/angular";
import {NgComponentOutlet} from "@angular/common";
import { DomSanitizer, SafeHtml } from '@angular/platform-browser';

@Component({
  selector: 'app-filetree',
  standalone: true,
  imports: [ClrTreeViewModule, NgComponentOutlet,],
  templateUrl: './filetree.component.html',
  styleUrl: './filetree.component.css'
})
export class FiletreeComponent implements OnInit{

  @Input() pluginData: Plugin[] | undefined;

  pluginsHTML: string = ""

  constructor(private sanitizer: DomSanitizer) {}

  ngOnInit() {
    if (this.pluginData != undefined) {
      this.parsePlugins(this.pluginData)

      console.log(this.trustedPluginsHTML)

    }
  }

  get trustedPluginsHTML(): SafeHtml{
    return this.sanitizer.bypassSecurityTrustHtml(this.pluginsHTML)
  }

  parsePlugins(pluginData: Plugin[]) {

    pluginData?.forEach(p => {

      this.pluginsHTML += `<clr-tree-node [clrExpanded]="${p.isdir}">${p.name}`

     if (p.children.length > 0) {
        this.parsePlugins(p.children)
      }

     this.pluginsHTML += `</clr-tree-node>`

    })

  }

  protected readonly ClrTreeNode = ClrTreeNode;
}
