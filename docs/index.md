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