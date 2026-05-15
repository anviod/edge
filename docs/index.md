---
layout: landing
title: EdgeX 项目文档
description: EdgeX 项目的完整文档

---

<section class="hero-section">
  <div class="shell shell--wide">
    <div class="hero-banner">
      <div class="hero-grid">
        <div class="hero-copy">
          <div class="eyebrow">Industrial Edge Documentation</div>
          <h1>EdgeX 文档</h1>
          <p>面向工业现场接入、边缘计算与北向集成的文档入口，集中提供 API、架构、部署、驱动适配、测试验证和运维资料。</p>
          <div class="hero-actions">
            <a class="button-link button-link--primary" href="guide/产品说明.html">产品介绍</a>
            <a class="button-link button-link--secondary" href="guide/USER_MANUAL.html">用户手册</a>
            <a class="button-link button-link--secondary" href="api/index.html">API 文档</a>
          </div>
          <div class="hero-metrics">
            <div class="metric-card">
              <strong>10+</strong>
              <span>核心文档分区</span>
            </div>
            <div class="metric-card">
              <strong>50+</strong>
              <span>文档数量</span>
            </div>
            <div class="metric-card">
              <strong>V0.0.1</strong>
              <span>当前版本</span>
            </div>
          </div>
        </div>
        <aside class="hero-panel">
          <p class="hero-panel__label">最近更新</p>
          <ul>
            <li><strong>2026年5月</strong>：新增 S7 协议支持</li>
            <li>支持 S7-200Smart/1200/1500/300/400 全系列 PLC</li>
            <li>AGReadMulti 批量读取优化，单次最多读取 20 个数据项</li>
            <li>详细实现请参阅<a href="drivers/PLC_S7.html">S7 协议文档</a></li>
          </ul>
        </aside>
      </div>
    </div>
  </div>
</section>

<section class="landing-section">
  <div class="shell shell--wide">
    <div class="wide-panel" style="background: linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 50%, #1a1a1a 100%); border: 1px solid #444; border-radius: 12px; padding: 35px; box-shadow: 0 4px 20px rgba(0,0,0,0.3);">
      <div class="section-kicker" style="color: #00d4ff;">计划方向</div>
      <h2 style="color: #ffffff; margin-bottom: 15px;">即将推出</h2>
      <p style="color: #b8c5d6; margin-bottom: 25px;">展示正在规划和开发中的功能，预计发布时间仅供参考。</p>
      
      <div style="display: flex; align-items: center; gap: 15px; margin-bottom: 20px; flex-wrap: wrap;">
        <span style="background: linear-gradient(135deg, #00d4ff, #0099cc); color: white; padding: 6px 16px; border-radius: 20px; font-size: 14px; font-weight: 600;">Q3 2026</span>
        <a href="drivers/index.html" style="color: #00d4ff; text-decoration: none; font-weight: 500; border-bottom: 2px solid transparent; transition: border-color 0.3s;">EtherNet/IP 驱动支持</a>
        <a href="https://github.com/anviod/logix" target="_blank" style="color: #00d4ff; text-decoration: none; font-weight: 500; border-bottom: 2px solid transparent; transition: border-color 0.3s;">logix 库</a>
        <a href="TODO/EtherNet_IP驱动真实通信实现方案.html" style="color: #ffd700; text-decoration: none; font-weight: 500; border-bottom: 2px solid transparent; transition: border-color 0.3s;">实现方案文档</a>
      </div>
      
      <div style="grid-template-columns: 1fr 1fr; gap: 30px; display: grid;">
        <div>
          <h3 style="color: #00d4ff; margin-bottom: 15px; font-size: 18px;">协议介绍</h3>
          <p style="color: #b8c5d6; font-size: 14px; line-height: 1.8;">
            <strong style="color: #ffffff;">EtherNet/IP</strong> 是工业以太网协议，基于 CIP (Common Industrial Protocol) 技术，广泛应用于 Allen-Bradley PLC 设备。支持 ControlLogix、CompactLogix、Micro800 等全系列 Rockwell PLC。
          </p>
        </div>
        
        <div>
          <h3 style="color: #00d4ff; margin-bottom: 15px; font-size: 18px;">技术栈</h3>
          <ul style="color: #b8c5d6; font-size: 14px; line-height: 1.8; list-style-type: none; padding-left: 0;">
            <li style="padding-left: 20px; position: relative;">
              <span style="position: absolute; left: 0; color: #00d4ff;">•</span>
              <strong style="color: #ffffff;">Go</strong> + <a href="https://github.com/anviod/logix" target="_blank" style="color: #00d4ff;">logix</a> 库
            </li>
            <li style="padding-left: 20px; position: relative;">
              <span style="position: absolute; left: 0; color: #00d4ff;">•</span>
              Vue 3 + Arco Design (前端)
            </li>
            <li style="padding-left: 20px; position: relative;">
              <span style="position: absolute; left: 0; color: #00d4ff;">•</span>
              TCP/IP 通信协议
            </li>
          </ul>
        </div>
      </div>
      
      <div style="margin-top: 25px; padding-top: 20px; border-top: 1px solid rgba(255,255,255,0.1);">
        <h3 style="color: #00d4ff; margin-bottom: 15px; font-size: 18px;">核心功能</h3>
        <ul style="color: #b8c5d6; font-size: 14px; line-height: 1.8; display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 10px;">
          <li style="padding-left: 20px; position: relative;">
            <span style="position: absolute; left: 0; color: #00ff88;">✓</span>
            基于 logix 库的真实 TCP 通信
          </li>
          <li style="padding-left: 20px; position: relative;">
            <span style="position: absolute; left: 0; color: #00ff88;">✓</span>
            Tag 地址格式批量读取与单点写入
          </li>
          <li style="padding-left: 20px; position: relative;">
            <span style="position: absolute; left: 0; color: #00ff88;">✓</span>
            支持 bool、sint、int、dint、real、string 数据类型
          </li>
          <li style="padding-left: 20px; position: relative;">
            <span style="position: absolute; left: 0; color: #00ff88;">✓</span>
            批量读取优化，减少网络往返
          </li>
          <li style="padding-left: 20px; position: relative;">
            <span style="position: absolute; left: 0; color: #00ff88;">✓</span>
            自动重连、心跳保活、健康状态检测
          </li>
          <li style="padding-left: 20px; position: relative;">
            <span style="position: absolute; left: 0; color: #00ff88;">✓</span>
            连接指标统计与监控
          </li>
        </ul>
      </div>
      
      <div style="margin-top: 25px; padding-top: 20px; border-top: 1px solid rgba(255,255,255,0.1);">
        <h3 style="color: #00d4ff; margin-bottom: 15px; font-size: 18px;">支持 PLC 系列</h3>
        <div style="display: flex; flex-wrap: wrap; gap: 10px;">
          <span style="background: rgba(0,212,255,0.15); color: #00d4ff; padding: 8px 16px; border-radius: 8px; font-size: 14px; border: 1px solid rgba(0,212,255,0.3);">ControlLogix</span>
          <span style="background: rgba(0,212,255,0.15); color: #00d4ff; padding: 8px 16px; border-radius: 8px; font-size: 14px; border: 1px solid rgba(0,212,255,0.3);">CompactLogix</span>
          <span style="background: rgba(0,212,255,0.15); color: #00d4ff; padding: 8px 16px; border-radius: 8px; font-size: 14px; border: 1px solid rgba(0,212,255,0.3);">Micro800</span>
          <span style="background: rgba(0,212,255,0.15); color: #00d4ff; padding: 8px 16px; border-radius: 8px; font-size: 14px; border: 1px solid rgba(0,212,255,0.3);">SLC 500</span>
          <span style="background: rgba(0,212,255,0.15); color: #00d4ff; padding: 8px 16px; border-radius: 8px; font-size: 14px; border: 1px solid rgba(0,212,255,0.3);">PLC-5</span>
        </div>
      </div>
    </div>
  </div>
</section>

<section class="landing-section">
  <div class="shell shell--wide">
    <div class="section-heading">
      <div>
        <div class="section-kicker">核心文档入口</div>
        <h2>快速导航</h2>
        <p>按使用场景整理常用资料，适合从接口联调、系统部署、驱动开发、架构评审和日常运维等任务。</p>
      </div>
    </div>

    <div class="cards-grid">
      <article class="feature-card">
        <span class="feature-card__tag">指南</span>
        <h3>产品与手册</h3>
        <p>产品介绍、用户手册和最佳实践，帮助您快速了解和使用 EdgeX 网关。</p>
        <div class="feature-card__links">
          <a class="mini-link" href="guide/产品说明.html">产品说明</a>
          <a class="mini-link" href="guide/USER_MANUAL.html">用户手册</a>
          <a class="mini-link" href="guide/EDGE_COMPUTING_BEST_PRACTICES.html">最佳实践</a>
        </div>
      </article>

      <article class="feature-card">
        <span class="feature-card__tag">API</span>
        <h3>接口文档</h3>
        <p>完整的 API 接口文档，涵盖认证、通道设备管理、边缘计算、北向配置和系统管理。</p>
        <div class="feature-card__links">
          <a class="mini-link" href="api/index.html">API 索引</a>
          <a class="mini-link" href="api/Authentication_CN.html">认证 API</a>
          <a class="mini-link" href="api/Channel_Device_Management_CN.html">设备管理</a>
        </div>
      </article>

      <article class="feature-card">
        <span class="feature-card__tag">架构</span>
        <h3>架构设计</h3>
        <p>聚焦三级架构、后端重构、状态机和数据源/动作设计，适合做方案评审和版本规划。</p>
        <div class="feature-card__links">
          <a class="mini-link" href="architecture/index.html">架构总览</a>
          <a class="mini-link" href="architecture/ARCHITECTURE_V2.html">架构 V2</a>
          <a class="mini-link" href="architecture/STATE_MACHINE_API.html">状态机 API</a>
        </div>
      </article>

      <article class="feature-card">
        <span class="feature-card__tag">驱动</span>
        <h3>设备驱动</h3>
        <p>覆盖 BACnet、OPC UA、Modbus、S7 的设计、测试、优化和故障分析，方便按协议深入。</p>
        <div class="feature-card__links">
          <a class="mini-link" href="drivers/index.html">驱动总览</a>
          <a class="mini-link" href="drivers/BACnet_设计说明.html">BACnet</a>
          <a class="mini-link" href="drivers/PLC_S7.html">S7 协议</a>
        </div>
      </article>

      <article class="feature-card">
        <span class="feature-card__tag">边缘</span>
        <h3>边缘计算</h3>
        <p>从基础功能到高阶能力，再到首页监控、功能走查、逻辑图和拓扑图。</p>
        <div class="feature-card__links">
          <a class="mini-link" href="edge/index.html">边缘计算总览</a>
          <a class="mini-link" href="edge/边缘计算基础功能.html">基础功能</a>
          <a class="mini-link" href="edge/边缘计算高阶功能.html">高阶功能</a>
        </div>
      </article>

      <article class="feature-card">
        <span class="feature-card__tag">部署</span>
        <h3>部署与交付</h3>
        <p>集成指南、快速启动、验收测试和验证报告，便于团队走完整交付链路。</p>
        <div class="feature-card__links">
          <a class="mini-link" href="deployment/index.html">部署总览</a>
          <a class="mini-link" href="deployment/INTEGRATION_GUIDE.html">集成指南</a>
          <a class="mini-link" href="deployment/QUICK_REFERENCE.html">快速参考</a>
        </div>
      </article>

      <article class="feature-card">
        <span class="feature-card__tag">北向</span>
        <h3>北向数据</h3>
        <p>聚焦 MQTT 数据上下行格式，适合联调和平台对接阶段。</p>
        <div class="feature-card__links">
          <a class="mini-link" href="northbound/index.html">北向总览</a>
          <a class="mini-link" href="northbound/MQTT数据上下行格式.html">MQTT 格式</a>
        </div>
      </article>

      <article class="feature-card">
        <span class="feature-card__tag">运维</span>
        <h3>系统运维</h3>
        <p>面向运行期稳定性，覆盖系统设置、网络、认证、数据库集成和运维手册。</p>
        <div class="feature-card__links">
          <a class="mini-link" href="operations/index.html">运维总览</a>
          <a class="mini-link" href="operations/边缘网关系统设置.html">系统设置</a>
          <a class="mini-link" href="operations/运维手册_BACnet.html">BACnet 运维</a>
        </div>
      </article>

      <article class="feature-card">
        <span class="feature-card__tag">测试</span>
        <h3>测试验证</h3>
        <p>测试矩阵、验收测试和回归验证方案，确保系统质量。</p>
        <div class="feature-card__links">
          <a class="mini-link" href="testing/index.html">测试总览</a>
          <a class="mini-link" href="testing/VERIFICATION_REPORT.html">验证报告</a>
          <a class="mini-link" href="testing/压力测试报告.html">压力测试</a>
        </div>
      </article>
    </div>
  </div>
</section>

<section class="landing-section">
  <div class="shell shell--wide">
    <div class="wide-panel">
      <div class="section-kicker">快速搜索</div>
      <h2>文档索引</h2>
      <p>按关键词快速查找相关文档：</p>
      <div class="quick-links">
        <a href="guide/产品说明.html">产品介绍</a>
        <a href="guide/USER_MANUAL.html">用户手册</a>
        <a href="api/index.html">API 文档</a>
        <a href="architecture/ARCHITECTURE_V2.html">架构设计</a>
        <a href="drivers/BACnet_设计说明.html">BACnet</a>
        <a href="drivers/MODBUS_OPTIMIZATION.html">Modbus</a>
        <a href="drivers/PLC_S7.html">S7</a>
        <a href="edge/边缘计算基础功能.html">边缘计算</a>
        <a href="northbound/MQTT数据上下行格式.html">MQTT</a>
        <a href="deployment/INTEGRATION_GUIDE.html">集成指南</a>
        <a href="operations/运维手册_BACnet.html">运维手册</a>
        <a href="testing/VERIFICATION_REPORT.html">验证报告</a>
      </div>
    </div>
  </div>
</section>

<section class="landing-section">
  <div class="shell shell--wide">
    <div class="section-heading">
      <div>
        <div class="section-kicker">完成的核心文档</div>
        <h2>智能采集优化系列</h2>
        <p>涵盖智能画像、影子设备、RTT/MTU/Gap 管理器等核心组件的设计与实现文档。</p>
      </div>
    </div>

    <div class="cards-grid cards-grid--two">
      <article class="feature-card">
        <span class="feature-card__tag">架构</span>
        <h3>项目现状分析</h3>
        <p>项目整体现状分析，为后续优化提供基础参考。</p>
        <div class="feature-card__links">
          <a class="mini-link" href="architecture/1. 项目现状分析.html">查看文档</a>
        </div>
      </article>

      <article class="feature-card">
        <span class="feature-card__tag">设计</span>
        <h3>智能画像方案</h3>
        <p>设备智能画像方案设计，支持自适应采集参数学习。</p>
        <div class="feature-card__links">
          <a class="mini-link" href="architecture/2. 智能画像方案设计.html">设计文档</a>
          <a class="mini-link" href="architecture/2. 智能画像方案设计_测试文档.html">测试文档</a>
        </div>
      </article>

      <article class="feature-card">
        <span class="feature-card__tag">设计</span>
        <h3>核心结构体定义</h3>
        <p>定义系统核心数据结构和接口规范。</p>
        <div class="feature-card__links">
          <a class="mini-link" href="architecture/3. 核心结构体定义.html">设计文档</a>
          <a class="mini-link" href="architecture/3. 核心结构体定义_测试文档.html">测试文档</a>
        </div>
      </article>

      <article class="feature-card">
        <span class="feature-card__tag">设计</span>
        <h3>核心设计</h3>
        <p>系统核心功能模块的详细设计文档。</p>
        <div class="feature-card__links">
          <a class="mini-link" href="architecture/4. 核心设计.html">设计文档</a>
          <a class="mini-link" href="architecture/4. 核心设计_测试文档.html">测试文档</a>
        </div>
      </article>

      <article class="feature-card">
        <span class="feature-card__tag">架构</span>
        <h3>实现架构</h3>
        <p>系统整体实现架构设计。</p>
        <div class="feature-card__links">
          <a class="mini-link" href="architecture/5. 实现架构.html">查看文档</a>
        </div>
      </article>

      <article class="feature-card">
        <span class="feature-card__tag">设计</span>
        <h3>影子设备设计</h3>
        <p>影子设备系统设计，支持数据一致性与快速恢复。</p>
        <div class="feature-card__links">
          <a class="mini-link" href="architecture/6. 影子设备设计.html">设计文档</a>
          <a class="mini-link" href="architecture/6. 影子设备设计_测试文档.html">测试文档</a>
          <a class="mini-link" href="architecture/影子设备与采集优化集成测试文档.html">集成测试</a>
          <a class="mini-link" href="architecture/影子设备系统联动关系文档.html">联动关系</a>
        </div>
      </article>

      <article class="feature-card">
        <span class="feature-card__tag">运维</span>
        <h3>边缘运维与设备替换</h3>
        <p>边缘环境下的运维策略与设备替换方案。</p>
        <div class="feature-card__links">
          <a class="mini-link" href="architecture/7. 边缘运维与设备替换.html">查看文档</a>
        </div>
      </article>

      <article class="feature-card">
        <span class="feature-card__tag">实现</span>
        <h3>RTT 管理器</h3>
        <p>基于 EWMA 算法的 RTT 延迟监测与超时自适应。</p>
        <div class="feature-card__links">
          <a class="mini-link" href="architecture/8. RTT管理器实现.html">实现文档</a>
          <a class="mini-link" href="architecture/8. RTT管理器实现_测试文档.html">测试文档</a>
        </div>
      </article>

      <article class="feature-card">
        <span class="feature-card__tag">实现</span>
        <h3>MTU 管理器</h3>
        <p>自动探测设备最大传输单元，优化批量读取效率。</p>
        <div class="feature-card__links">
          <a class="mini-link" href="architecture/9. MTU管理器实现.html">实现文档</a>
          <a class="mini-link" href="architecture/9. MTU管理器实现_测试文档.html">测试文档</a>
        </div>
      </article>

      <article class="feature-card">
        <span class="feature-card__tag">实现</span>
        <h3>Gap 优化器</h3>
        <p>基于设备负载动态调整通信间隔，优化总线效率。</p>
        <div class="feature-card__links">
          <a class="mini-link" href="architecture/10. Gap优化器实现.html">实现文档</a>
          <a class="mini-link" href="architecture/10. Gap优化器实现_测试文档.html">测试文档</a>
        </div>
      </article>
    </div>
  </div>
</section>

<section class="landing-section">
  <div class="shell shell--wide">
    <div class="section-heading">
      <div>
        <div class="section-kicker">最近更新</div>
        <h2>更新记录</h2>
      </div>
    </div>

    <div class="cards-grid cards-grid--two">
      <article class="feature-card">
        <span class="feature-card__tag">2026-02-25</span>
        <h3>点位管理增强</h3>
        <ul>
          <li>实现点位批量删除功能</li>
          <li>支持基于搜索关键词和质量状态的响应式实时过滤</li>
          <li>Modbus 稳定性优化：增加非法数据地址自动检测与 24 小时长冷却机制</li>
        </ul>
      </article>

      <article class="feature-card">
        <span class="feature-card__tag">2026-02-24</span>
        <h3>TCP 链路深度监控</h3>
        <ul>
          <li>增加本地 IP:端口、远程 IP:端口、链接时长及最后断开时间显示</li>
          <li>前端显示优化为直观的「本地 -> 远程」连接模式</li>
          <li>UI 对话框宽度增加 20%，提升信息展示密度</li>
        </ul>
      </article>

      <article class="feature-card">
        <span class="feature-card__tag">2026-02-20</span>
        <h3>Modbus 智能优化</h3>
        <ul>
          <li>智能 MTU 探测：自动探测并保存从站支持的最大寄存器数量</li>
          <li>指数退避重连：优化连接策略，避免网络抖动时的频繁重连尝试</li>
        </ul>
      </article>
    </div>
  </div>
</section>