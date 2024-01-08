import {Component, Input, OnInit} from '@angular/core';
import {Plugin} from "../../../model/plugins";
import {ClrTreeNode, ClrTreeViewModule} from "@clr/angular";
import {NgComponentOutlet} from "@angular/common";
import {DomSanitizer, SafeHtml} from '@angular/platform-browser';

@Component({
  selector: 'app-filetree',
  standalone: true,
  imports: [ClrTreeViewModule, NgComponentOutlet,],
  templateUrl: './filetree.component.html',
  styleUrl: './filetree.component.css'
})
export class FiletreeComponent implements OnInit {

  @Input() pluginData: Plugin[] | undefined;

  folderIcon = "<svg xmlns=\"http://www.w3.org/2000/svg\" fill=\"none\" viewBox=\"0 0 24 24\" stroke-width=\"1.5\" stroke=\"currentColor\" class=\"w-6 h-6\">\n" +
    "  <path stroke-linecap=\"round\" stroke-linejoin=\"round\" d=\"M2.25 12.75V12A2.25 2.25 0 0 1 4.5 9.75h15A2.25 2.25 0 0 1 21.75 12v.75m-8.69-6.44-2.12-2.12a1.5 1.5 0 0 0-1.061-.44H4.5A2.25 2.25 0 0 0 2.25 6v12a2.25 2.25 0 0 0 2.25 2.25h15A2.25 2.25 0 0 0 21.75 18V9a2.25 2.25 0 0 0-2.25-2.25h-5.379a1.5 1.5 0 0 1-1.06-.44Z\" />\n" +
    "</svg>\n"

  pluginsHTML: string = ""

  constructor(private sanitizer: DomSanitizer) {
  }

  ngOnInit() {
    if (this.pluginData != undefined) {
      this.parsePlugins(this.pluginData)

      console.log(this.trustedPluginsHTML)

    }
  }

  get trustedPluginsHTML(): SafeHtml {
    return this.sanitizer.bypassSecurityTrustHtml(this.pluginsHTML)
  }

  parsePlugins(pluginData: Plugin[]) {

    pluginData?.forEach(p => {
      if (p.name !== '.DS_Store') {

        this.pluginsHTML += (p.isdir) ? `<li><img style="width: 15px; height:auto;" src="../../../../assets/folder.png" alt=""/> ${p.name}` : `<li>${p.name}`

        if (p.children.length > 0) {
          this.pluginsHTML += `<ul style="list-style-type: none;">`
          this.parsePlugins(p.children)
          this.pluginsHTML += `</ul>`
        }
        this.pluginsHTML += `</li>`
      }

    })

  }

  protected readonly ClrTreeNode = ClrTreeNode;
}
